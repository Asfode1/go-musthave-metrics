package main

import (
	"github.com/Asfode1/go-musthave-metrics/internal/agent"
	"github.com/Asfode1/go-musthave-metrics/internal/config"
)

func main() {
	// Парсим конфигурацию из командной строки
	cfg := config.ParseAgentConfig()

	// Создаем и запускаем агент
	runner := agent.NewRunnerFromAgentConfig(cfg)
	runner.Start()

	// Чтобы основная горутина не завершилась и агент продолжал работать
	select {}
}
