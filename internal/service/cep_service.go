package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cep-api/config"
	"cep-api/internal/model"
	"cep-api/internal/ports"
	"cep-api/pkg/logger"
)

type CEPServiceImpl struct {
	cfg    *config.Config
	client *http.Client
}

func NewCEPService(cfg *config.Config) *CEPServiceImpl {
	return &CEPServiceImpl{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout + 500*time.Millisecond,
		},
	}
}

func (s *CEPServiceImpl) LookupCEP(ctx context.Context, cep string) (*model.CEPResult, error) {
	type result struct {
		provider string
		data     json.RawMessage
		err      error
		notFound bool
	}

	logger.Infof(ctx, "CEPService.LookupCEP", "iniciando consulta para CEP %s", cep)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan result, 2)

	go func() {
		url := fmt.Sprintf(s.cfg.BrasilAPIURL, cep)
		data, err := s.fetchAPI(ctx, url)
		notFound := err != nil && err.Error() == "status 404"
		ch <- result{provider: "BrasilAPI", data: data, err: err, notFound: notFound}
	}()

	go func() {
		url := fmt.Sprintf(s.cfg.ViaCEPURL, cep)
		data, err := s.fetchAPI(ctx, url)
		notFound := false
		if err == nil && isViaCEPError(data) {
			err = fmt.Errorf("CEP não encontrado")
			notFound = true
		}
		ch <- result{provider: "ViaCEP", data: data, err: err, notFound: notFound}
	}()

	select {
	case r := <-ch:
		if r.err == nil {
			logger.Infof(ctx, "CEPService.LookupCEP", "CEP %s retornado por %s", cep, r.provider)
			return &model.CEPResult{Provider: r.provider, Data: r.data}, nil
		}
		logger.Warnf(ctx, "CEPService.LookupCEP", "falha em %s para CEP %s: %s", r.provider, cep, r.err.Error())
		// Primeira falhou, aguarda a segunda
		select {
		case r2 := <-ch:
			if r2.err == nil {
				logger.Infof(ctx, "CEPService.LookupCEP", "CEP %s retornado por %s", cep, r2.provider)
				return &model.CEPResult{Provider: r2.provider, Data: r2.data}, nil
			}
			logger.Warnf(ctx, "CEPService.LookupCEP", "falha em %s para CEP %s: %s", r2.provider, cep, r2.err.Error())
			if r.notFound && r2.notFound {
				logger.Warnf(ctx, "CEPService.LookupCEP", "CEP %s não encontrado em nenhuma API", cep)
				return nil, ports.ErrNotFound
			}
			logger.Error(ctx, "CEPService.LookupCEP", "ambas as APIs falharam para CEP "+cep, ports.ErrBothFailed)
			return nil, ports.ErrBothFailed
		case <-time.After(s.cfg.Timeout):
			logger.Warnf(ctx, "CEPService.LookupCEP", "timeout aguardando segunda API para CEP %s", cep)
			return nil, ports.ErrTimeout
		}
	case <-time.After(s.cfg.Timeout):
		logger.Warnf(ctx, "CEPService.LookupCEP", "timeout: nenhuma API respondeu para CEP %s", cep)
		return nil, ports.ErrTimeout
	}
}

func isViaCEPError(data json.RawMessage) bool {
	var v struct {
		Erro string `json:"erro"`
	}
	return json.Unmarshal(data, &v) == nil && v.Erro == "true"
}

func (s *CEPServiceImpl) fetchAPI(ctx context.Context, url string) (json.RawMessage, error) {
	logger.Debugf(ctx, "CEPService.fetchAPI", "chamando %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(body), nil
}
