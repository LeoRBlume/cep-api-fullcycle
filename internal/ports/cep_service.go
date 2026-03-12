package ports

import (
	"context"

	"cep-api/internal/model"
)

type CEPService interface {
	LookupCEP(ctx context.Context, cep string) (*model.CEPResult, error)
}
