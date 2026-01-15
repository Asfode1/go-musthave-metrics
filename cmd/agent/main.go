package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Asfode1/go-musthave-metrics/internal/agent"
)

const (
	defaultServerURL      = "http://localhost:8080"
	defaultPollInterval   = 2  // секунды
	defaultReportInterval = 10 // секунды
)

func main() {
	// Чтобы получить конфигурацию из командной строки
	serverURL := flag.String("a", defaultServerURL, "HTTP server address")
	pollIntervalSec := flag.Int("p", defaultPollInterval, "Poll interval in seconds")
	reportIntervalSec := flag.Int("r", defaultReportInterval, "Report interval in seconds")
	
	// Чтобы валидировать только разрешенные флаги
	knownFlags := map[string]bool{
		"a": true,
		"p": true,
		"r": true,
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

	// Чтобы предотвратить некорректные значения интервалов
	if *pollIntervalSec <= 0 {
		fmt.Fprintf(os.Stderr, "Error: poll interval must be positive, got %d\n", *pollIntervalSec)
		os.Exit(1)
	}
	if *reportIntervalSec <= 0 {
		fmt.Fprintf(os.Stderr, "Error: report interval must be positive, got %d\n", *reportIntervalSec)
		os.Exit(1)
	}

	// Чтобы использовать интервалы в стандартном формате Go
	pollInterval := time.Duration(*pollIntervalSec) * time.Second
	reportInterval := time.Duration(*reportIntervalSec) * time.Second

	collector := agent.NewCollector()
	client := agent.NewClient(*serverURL)

	// Чтобы асинхронно передавать метрики между горутинами без блокировок
	metricsChan := make(chan []agent.MetricValue, 1)

	// Чтобы собирать метрики с заданной периодичностью независимо от отправки
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for range ticker.C {
			metrics := collector.Collect()
			// Чтобы не блокировать сбор метрик при переполнении канала
			select {
			case metricsChan <- metrics:
			default:
				// Чтобы всегда иметь актуальные метрики, удаляем старые при переполнении
				<-metricsChan
				metricsChan <- metrics
			}
		}
	}()

	// Чтобы отправлять метрики на сервер с заданной периодичностью независимо от сбора
	go func() {
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()

		for range ticker.C {
			// Чтобы использовать актуальные метрики для отправки
			var metrics []agent.MetricValue
			select {
			case metrics = <-metricsChan:
			default:
				// Чтобы гарантировать отправку метрик даже если канал пуст
				metrics = collector.Collect()
			}

			// Чтобы сервер получил актуальные данные о состоянии системы
			if err := client.SendMetrics(metrics); err != nil {
				log.Printf("Failed to send metrics: %v", err)
			} else {
				log.Printf("Successfully sent %d metrics", len(metrics))
			}
		}
	}()

	// Чтобы агент продолжал работать до явного завершения
	log.Println("Agent started")
	log.Printf("Poll interval: %v", pollInterval)
	log.Printf("Report interval: %v", reportInterval)
	log.Printf("Server URL: %s", *serverURL)

	// Чтобы основная горутина не завершилась и агент продолжал работать
	select {}
}
