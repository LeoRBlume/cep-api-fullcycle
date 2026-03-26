package main

import (
	"context"

	"cep-api/internal/app"
	"cep-api/pkg/logger"
)

func main() {
	logger.Setup(logger.Config{
		ServiceName: "cep-api",
		Level:       logger.LevelInfo,
	})

	if err := app.Run(); err != nil {
		logger.Fatal(context.Background(), "main", "falha ao iniciar aplicação", err)
	}
}
