package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"cep-api/internal/model"
	"cep-api/internal/ports"

	"github.com/gin-gonic/gin"
)

type CEPController struct {
	service ports.CEPService
}

func NewCEPController(service ports.CEPService) *CEPController {
	return &CEPController{service: service}
}

func (c *CEPController) LookupCEP(ctx *gin.Context) {
	cep := ctx.Param("cep")

	result, err := c.service.LookupCEP(ctx.Request.Context(), cep)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			fmt.Printf("\n[TIMEOUT] Nenhuma API respondeu dentro do prazo para o CEP: %s\n", cep)
			ctx.JSON(http.StatusGatewayTimeout, gin.H{
				"error": err.Error(),
			})
			return
		}

		fmt.Printf("\n[ERRO] Falha ao consultar CEP %s: %s\n", cep, err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "erro interno ao consultar CEP",
		})
		return
	}

	printResult(cep, result)

	ctx.JSON(http.StatusOK, result)
}

func printResult(cep string, result *model.CEPResult) {
	fmt.Println("\n========================================")
	fmt.Printf("CEP Consultado: %s\n", cep)
	fmt.Printf("Resposta de: %s\n", result.Provider)
	fmt.Println("----------------------------------------")

	var pretty map[string]any
	if err := json.Unmarshal(result.Data, &pretty); err == nil {
		for k, v := range pretty {
			fmt.Printf("  %s: %v\n", k, v)
		}
	} else {
		fmt.Printf("  %s\n", string(result.Data))
	}

	fmt.Println("========================================")
}
