package app

import (
	"cep-api/config"
	"cep-api/internal/controller"
	"cep-api/internal/router"
	"cep-api/internal/service"
)

func Run() error {
	cfg := config.NewDefaultConfig()

	svc := service.NewCEPService(cfg)
	ctrl := controller.NewCEPController(svc)
	r := router.SetupRouter(ctrl)

	return r.Run(cfg.Port)
}
