package router

import (
	"cep-api/internal/controller"

	"github.com/gin-gonic/gin"
)

func SetupRouter(ctrl *controller.CEPController) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/cep/:cep", ctrl.LookupCEP)
	return r
}
