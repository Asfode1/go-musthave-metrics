package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultServerAddress     = "localhost:8080"
	defaultStoreIntervalSec  = 300
	defaultFileStoragePath   = "metrics-db.json"
	defaultRestoreOnStartup  = true
)

type ServerConfig struct {
	Address         string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
}

// ParseServerConfig читает конфигурацию сервера с приоритетом env > flag > default.
// ADDRESS, STORE_INTERVAL, FILE_STORAGE_PATH, RESTORE — переменные окружения.
// -a, -i, -f, -r — флаги.
func ParseServerConfig() *ServerConfig {
	var address string
	flag.StringVar(&address, "a", defaultServerAddress, "HTTP server address")
	var storeIntervalSec int
	flag.IntVar(&storeIntervalSec, "i", defaultStoreIntervalSec, "Store interval in seconds (0 = sync)")
	var fileStoragePath string
	flag.StringVar(&fileStoragePath, "f", defaultFileStoragePath, "File storage path")
	var restore bool
	flag.BoolVar(&restore, "r", defaultRestoreOnStartup, "Restore metrics from file on startup")

	validateServerFlags()

	flag.Parse()

	if len(flag.Args()) > 0 {
		log.Fatalf("Error: unknown arguments: %v\n", flag.Args())
	}

	applyServerEnvOverrides(&address, &storeIntervalSec, &fileStoragePath, &restore)

	if storeIntervalSec < 0 {
		log.Fatalf("Error: STORE_INTERVAL must be >= 0 seconds, got %d\n", storeIntervalSec)
	}

	if strings.TrimSpace(fileStoragePath) == "" {
		log.Fatalf("Error: FILE_STORAGE_PATH is empty\n")
	}

	return &ServerConfig{
		Address:         address,
		StoreInterval:   time.Duration(storeIntervalSec) * time.Second,
		FileStoragePath: fileStoragePath,
		Restore:         restore,
	}
}

func applyServerEnvOverrides(address *string, storeIntervalSec *int, fileStoragePath *string, restore *bool) {
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		v = strings.TrimSpace(v)
		if v == "" {
			log.Fatalf("Error: ADDRESS env var is empty\n")
		}
		*address = v
	}

	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		v = strings.TrimSpace(v)
		if v == "" {
			log.Fatalf("Error: STORE_INTERVAL env var is empty\n")
		}
		sec, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("Error: STORE_INTERVAL must be integer seconds, got %q\n", v)
		}
		*storeIntervalSec = sec
	}

	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		v = strings.TrimSpace(v)
		if v == "" {
			log.Fatalf("Error: FILE_STORAGE_PATH env var is empty\n")
		}
		*fileStoragePath = v
	}

	if v, ok := os.LookupEnv("RESTORE"); ok {
		v = strings.TrimSpace(v)
		if v == "" {
			log.Fatalf("Error: RESTORE env var is empty\n")
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			log.Fatalf("Error: RESTORE must be true/false, got %q\n", v)
		}
		*restore = b
	}
}

func validateServerFlags() {
	knownFlags := map[string]bool{
		"a": true,
		"i": true,
		"f": true,
		"r": true,
	}

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if len(arg) > 1 && arg[0] == '-' {
			flagName := arg[1:]
			// корректно обработать флаги вида -a=value
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
}

