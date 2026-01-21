package config

import (
	"flag"
	"log"
	"os"
	"time"
)

const (
	DefaultServerURL      = "http://localhost:8080"
	DefaultPollInterval   = 2  // секунды
	DefaultReportInterval = 10 // секунды
)

// AgentConfig конфигурация агента
type AgentConfig struct {
	ServerURL      string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

// ParseAgentConfig парсит флаги командной строки и возвращает конфигурацию агента
func ParseAgentConfig() *AgentConfig {
	var serverURL string
	var pollIntervalSec int
	var reportIntervalSec int

	flag.StringVar(&serverURL, "a", DefaultServerURL, "HTTP server address")
	flag.IntVar(&pollIntervalSec, "p", DefaultPollInterval, "Poll interval in seconds")
	flag.IntVar(&reportIntervalSec, "r", DefaultReportInterval, "Report interval in seconds")

	validateFlags()

	flag.Parse()

	if len(flag.Args()) > 0 {
		log.Fatalf("Error: unknown arguments: %v\n", flag.Args())
	}

	if pollIntervalSec <= 0 {
		log.Fatalf("Error: poll interval must be positive, got %d\n", pollIntervalSec)
	}
	if reportIntervalSec <= 0 {
		log.Fatalf("Error: report interval must be positive, got %d\n", reportIntervalSec)
	}

	return &AgentConfig{
		ServerURL:      serverURL,
		PollInterval:   time.Duration(pollIntervalSec) * time.Second,
		ReportInterval: time.Duration(reportIntervalSec) * time.Second,
	}
}

// validateFlags проверяет, что используются только разрешенные флаги
func validateFlags() {
	knownFlags := map[string]bool{
		"a": true,
		"p": true,
		"r": true,
	}

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if len(arg) > 1 && arg[0] == '-' {
			flagName := arg[1:]
			// Чтобы корректно обработать флаги вида -a=value
			for j := 0; j < len(flagName); j++ {
				if flagName[j] == '=' {
					flagName = flagName[:j]
					break
				}
			}
			if !knownFlags[flagName] {
				log.Fatalf("Error: unknown flag: -%s\n", flagName)
			}
		}
	}
}
