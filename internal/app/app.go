package app

import (
	"context"
	"net/http"

	"cep-api/config"
	"cep-api/internal/controller"
	"cep-api/internal/router"
	"cep-api/internal/service"
	"cep-api/pkg/logger"
)

func Run() error {
	cfg := config.NewDefaultConfig()

	logger.Infof(context.Background(), "app.Run", "iniciando servidor na porta %s", cfg.Port)

	svc := service.NewCEPService(cfg)
	ctrl := controller.NewCEPController(svc)
	logger.Infof(context.Background(), "app.Run", "Iniciando router")
	r := router.SetupRouter(ctrl)

	return http.ListenAndServe(cfg.Port, logger.TraceMiddleware(r))
}
