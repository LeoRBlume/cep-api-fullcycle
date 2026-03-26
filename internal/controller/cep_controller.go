package controller

import (
	"net/http"
	"regexp"

	"cep-api/internal/ports"
	apperrors "cep-api/pkg/errors"
	"cep-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

var cepRegex = regexp.MustCompile(`^\d{8}$`)

type CEPController struct {
	service    ports.CEPService
	translator *apperrors.Translator
}

func NewCEPController(service ports.CEPService) *CEPController {
	translator := apperrors.NewTranslator().
		Register(ports.ErrNotFound, http.StatusNotFound).
		Register(ports.ErrTimeout, http.StatusGatewayTimeout).
		Register(ports.ErrBothFailed, http.StatusInternalServerError)

	return &CEPController{service: service, translator: translator}
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
		status := c.translator.Translate(err)
		logger.Errorf(reqCtx, "CEPController.LookupCEP", "erro ao consultar CEP %s", err, cep)
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	logger.Infof(reqCtx, "CEPController.LookupCEP", "CEP %s consultado com sucesso via %s", cep, result.Provider)
	ctx.JSON(http.StatusOK, result)
}
