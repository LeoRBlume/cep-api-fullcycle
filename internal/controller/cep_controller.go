package controller

import (
	"errors"
	"net/http"
	"regexp"

	"cep-api/internal/ports"
	"cep-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

var cepRegex = regexp.MustCompile(`^\d{8}$`)

type CEPController struct {
	service ports.CEPService
}

func NewCEPController(service ports.CEPService) *CEPController {
	return &CEPController{service: service}
}

func (c *CEPController) LookupCEP(ctx *gin.Context) {
	cep := ctx.Param("cep")
	reqCtx := ctx.Request.Context()

	if !cepRegex.MatchString(cep) {
		logger.Warnf(reqCtx, "CEPController.LookupCEP", "CEP inválido recebido: %s", cep)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "CEP inválido: deve conter 8 dígitos numéricos",
		})
		return
	}

	result, err := c.service.LookupCEP(reqCtx, cep)
	if err != nil {
		if errors.Is(err, ports.ErrTimeout) {
			logger.Warnf(reqCtx, "CEPController.LookupCEP", "timeout ao consultar CEP %s", cep)
			ctx.JSON(http.StatusGatewayTimeout, gin.H{
				"error": err.Error(),
			})
			return
		}

		logger.Errorf(reqCtx, "CEPController.LookupCEP", "erro ao consultar CEP %s", err, cep)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "erro interno ao consultar CEP",
		})
		return
	}

	logger.Infof(reqCtx, "CEPController.LookupCEP", "CEP %s consultado com sucesso via %s", cep, result.Provider)
	ctx.JSON(http.StatusOK, result)
}
