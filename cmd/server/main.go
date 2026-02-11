package main

import (
	"log"
	"net/http"

	"github.com/Asfode1/go-musthave-metrics/internal/config"
	"github.com/Asfode1/go-musthave-metrics/internal/handler"
	"github.com/Asfode1/go-musthave-metrics/internal/mw"
	"github.com/Asfode1/go-musthave-metrics/internal/service"
	"github.com/Asfode1/go-musthave-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
	// Конфигурация (env > flag > default)
	cfg := config.ParseServerConfig()

	// Чтобы хранить метрики в памяти для быстрого доступа
	memStorage := storage.NewMemStorage()

	// Persist: restore/save metrics to file
	persister := service.NewPersister(memStorage, cfg.FileStoragePath)
	if cfg.Restore {
		if err := persister.Load(); err != nil {
			log.Printf("Failed to restore metrics: %v", err)
		}
	}
	if cfg.StoreInterval > 0 {
		persister.StartPeriodic(cfg.StoreInterval, func(err error) {
			log.Printf("Failed to store metrics: %v", err)
		})
	}

	var saveFn func() error
	if cfg.StoreInterval == 0 {
		saveFn = persister.Save
	}

	// Чтобы обрабатывать HTTP запросы для работы с метриками
	metricsHandler := handler.NewMetricsHandlerWithSaver(memStorage, saveFn)

	// Чтобы использовать удобный роутер для маршрутизации запросов
	r := chi.NewRouter()

	// Логирование запросов/ответов через middleware (уровень Info)
	zlogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer func() { _ = zlogger.Sync() }()

	r.Use(mw.HTTPLogger(zlogger))
	r.Use(mw.Gzip)
	r.Use(chimw.Recoverer)

	// Чтобы обрабатывать различные типы запросов к метрикам
	r.Post("/update", metricsHandler.UpdateJSON)
	r.Post("/updates", metricsHandler.UpdateBatchJSON)
	r.Post("/value", metricsHandler.ValueJSON)
	r.Post("/update/{type}/{name}/{value}", metricsHandler.Update)
	r.Get("/value/{type}/{name}", metricsHandler.Value)

	// Чтобы начать принимать HTTP запросы от агентов
	server := &http.Server{
		Addr:    cfg.Address,
		Handler: r,
	}

	log.Printf("Server starting on http://%s", cfg.Address)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
