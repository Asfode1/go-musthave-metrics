package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Asfode1/go-musthave-metrics/internal/handler"
	"github.com/Asfode1/go-musthave-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	defaultAddress = "localhost:8080"
)

func main() {
	// Чтобы получить конфигурацию из командной строки
	address := flag.String("a", defaultAddress, "HTTP server address")

	// Чтобы валидировать только разрешенные флаги
	knownFlags := map[string]bool{
		"a": true,
	}

	// Чтобы предотвратить использование неизвестных флагов и обеспечить валидность конфигурации
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if len(arg) > 1 && arg[0] == '-' {
			// Извлекаем имя флага для проверки
			flagName := arg[1:]
			// Чтобы корректно обработать флаги вида -a=value
			for j := 0; j < len(flagName); j++ {
				if flagName[j] == '=' {
					flagName = flagName[:j]
					break
				}
			}
			if !knownFlags[flagName] {
				fmt.Fprintf(os.Stderr, "Error: unknown flag: -%s\n", flagName)
				os.Exit(1)
			}
		}
	}

	// Чтобы извлечь значения флагов из аргументов
	flag.Parse()

	// Чтобы предотвратить использование неизвестных аргументов
	if len(flag.Args()) > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		os.Exit(1)
	}

	// Чтобы хранить метрики в памяти для быстрого доступа
	memStorage := storage.NewMemStorage()

	// Чтобы обрабатывать HTTP запросы для работы с метриками
	metricsHandler := handler.NewMetricsHandler(memStorage)

	// Чтобы использовать удобный роутер для маршрутизации запросов
	r := chi.NewRouter()

	// Чтобы логировать все HTTP запросы для отладки и мониторинга
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Чтобы обрабатывать различные типы запросов к метрикам
	r.Post("/update/{type}/{name}/{value}", metricsHandler.Update)

	// Чтобы начать принимать HTTP запросы от агентов
	server := &http.Server{
		Addr:    *address,
		Handler: r,
	}

	log.Printf("Server starting on http://%s", *address)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
