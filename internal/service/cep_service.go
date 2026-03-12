package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"cep-api/config"
	"cep-api/internal/model"
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
	}

	ch := make(chan result, 2)

	go func() {
		url := fmt.Sprintf(s.cfg.BrasilAPIURL, cep)
		data, err := s.fetchAPI(ctx, url)
		ch <- result{provider: "BrasilAPI", data: data, err: err}
	}()

	go func() {
		url := fmt.Sprintf(s.cfg.ViaCEPURL, cep)
		data, err := s.fetchAPI(ctx, url)
		ch <- result{provider: "ViaCEP", data: data, err: err}
	}()

	select {
	case r := <-ch:
		if r.err == nil {
			return &model.CEPResult{Provider: r.provider, Data: r.data}, nil
		}
		// First failed, wait for second
		select {
		case r2 := <-ch:
			if r2.err == nil {
				return &model.CEPResult{Provider: r2.provider, Data: r2.data}, nil
			}
			return nil, errors.New("ambas as APIs falharam")
		case <-time.After(s.cfg.Timeout):
			return nil, errors.New("timeout: nenhuma API respondeu dentro de 1 segundo")
		}
	case <-time.After(s.cfg.Timeout):
		return nil, errors.New("timeout: nenhuma API respondeu dentro de 1 segundo")
	}
}

func (s *CEPServiceImpl) fetchAPI(ctx context.Context, url string) (json.RawMessage, error) {
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
