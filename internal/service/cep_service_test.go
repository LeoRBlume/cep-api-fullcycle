package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cep-api/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConfig(brasilURL, viacepURL string, timeout time.Duration) *config.Config {
	return &config.Config{
		BrasilAPIURL: brasilURL + "/%s",
		ViaCEPURL:    viacepURL + "/%s",
		Timeout:      timeout,
		Port:         ":8080",
	}
}

func TestLookupCEP_BrasilAPIRespondsFirst(t *testing.T) {
	brasilServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"cep":"01001000","state":"SP","city":"São Paulo"}`)
	}))
	defer brasilServer.Close()

	viacepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"cep":"01001-000","logradouro":"Praça da Sé"}`)
	}))
	defer viacepServer.Close()

	cfg := newTestConfig(brasilServer.URL, viacepServer.URL, 2*time.Second)
	svc := NewCEPService(cfg)

	result, err := svc.LookupCEP(context.Background(), "01001000")
	require.NoError(t, err)
	assert.Equal(t, "BrasilAPI", result.Provider)
	assert.Contains(t, string(result.Data), "01001000")
}

func TestLookupCEP_ViaCEPRespondsFirst(t *testing.T) {
	brasilServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"cep":"01001000","state":"SP"}`)
	}))
	defer brasilServer.Close()

	viacepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"cep":"01001-000","logradouro":"Praça da Sé"}`)
	}))
	defer viacepServer.Close()

	cfg := newTestConfig(brasilServer.URL, viacepServer.URL, 2*time.Second)
	svc := NewCEPService(cfg)

	result, err := svc.LookupCEP(context.Background(), "01001000")
	require.NoError(t, err)
	assert.Equal(t, "ViaCEP", result.Provider)
	assert.Contains(t, string(result.Data), "01001-000")
}

func TestLookupCEP_BothTimeout(t *testing.T) {
	brasilServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		fmt.Fprint(w, `{}`)
	}))
	defer brasilServer.Close()

	viacepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		fmt.Fprint(w, `{}`)
	}))
	defer viacepServer.Close()

	cfg := newTestConfig(brasilServer.URL, viacepServer.URL, 500*time.Millisecond)
	svc := NewCEPService(cfg)

	result, err := svc.LookupCEP(context.Background(), "99999999")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestLookupCEP_OneFailsOtherResponds(t *testing.T) {
	brasilServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer brasilServer.Close()

	viacepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"cep":"01001-000","logradouro":"Praça da Sé"}`)
	}))
	defer viacepServer.Close()

	cfg := newTestConfig(brasilServer.URL, viacepServer.URL, 2*time.Second)
	svc := NewCEPService(cfg)

	result, err := svc.LookupCEP(context.Background(), "01001000")
	require.NoError(t, err)
	assert.Equal(t, "ViaCEP", result.Provider)
}

func TestLookupCEP_BothFail(t *testing.T) {
	brasilServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer brasilServer.Close()

	viacepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer viacepServer.Close()

	cfg := newTestConfig(brasilServer.URL, viacepServer.URL, 2*time.Second)
	svc := NewCEPService(cfg)

	result, err := svc.LookupCEP(context.Background(), "00000000")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "falharam")
}
