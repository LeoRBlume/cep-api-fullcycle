package ports

import (
	"context"
	"errors"

	"cep-api/internal/model"
)

var (
	ErrTimeout    = errors.New("timeout: nenhuma API respondeu dentro de 1 segundo")
	ErrBothFailed = errors.New("ambas as APIs falharam")
)

type CEPService interface {
	LookupCEP(ctx context.Context, cep string) (*model.CEPResult, error)
}
