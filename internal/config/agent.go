package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
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

	applyAgentEnvOverrides(&serverURL, &pollIntervalSec, &reportIntervalSec)

	if pollIntervalSec <= 0 {
		log.Fatalf("Error: poll interval must be positive, got %d\n", pollIntervalSec)
	}
	if reportIntervalSec <= 0 {
		log.Fatalf("Error: report interval must be positive, got %d\n", reportIntervalSec)
	}

	return &AgentConfig{
		ServerURL:      normalizeAgentAddress(serverURL),
		PollInterval:   time.Duration(pollIntervalSec) * time.Second,
		ReportInterval: time.Duration(reportIntervalSec) * time.Second,
	}
}

func applyAgentEnvOverrides(serverURL *string, pollIntervalSec *int, reportIntervalSec *int) {
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		v = strings.TrimSpace(v)
		if v == "" {
			log.Fatalf("Error: ADDRESS env var is empty\n")
		}
		*serverURL = v
	}

	if v, ok := envPositiveIntSeconds("POLL_INTERVAL"); ok {
		*pollIntervalSec = v
	}
	if v, ok := envPositiveIntSeconds("REPORT_INTERVAL"); ok {
		*reportIntervalSec = v
	}
}

func envPositiveIntSeconds(name string) (int, bool) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return 0, false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		log.Fatalf("Error: %s env var is empty\n", name)
	}
	sec, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("Error: %s must be integer seconds, got %q\n", name, v)
	}
	if sec <= 0 {
		log.Fatalf("Error: %s must be positive seconds, got %d\n", name, sec)
	}
	return sec, true
}

// normalizeAgentAddress принимает ADDRESS в виде host:port или URL.
// Если схемы нет — добавляет http:// (это нужно для клиентских запросов агента).
func normalizeAgentAddress(address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return DefaultServerURL
	}
	if strings.Contains(address, "://") {
		return address
	}
	return "http://" + address
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
