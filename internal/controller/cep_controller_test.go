package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cep-api/internal/model"
	"cep-api/internal/ports"
	"cep-api/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupRouter(ctrl *CEPController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/cep/:cep", ctrl.LookupCEP)
	return r
}

func TestLookupCEP_BrasilAPI(t *testing.T) {
	mockService := mocks.NewCEPService(t)
	ctrl := NewCEPController(mockService)

	expectedData := json.RawMessage(`{"cep":"01001000","state":"SP","city":"São Paulo"}`)
	mockService.On("LookupCEP", mock.Anything, "01001000").Return(&model.CEPResult{
		Provider: "BrasilAPI",
		Data:     expectedData,
	}, nil)

	r := setupRouter(ctrl)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/cep/01001000", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.CEPResult
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "BrasilAPI", response.Provider)
}

func TestLookupCEP_ViaCEP(t *testing.T) {
	mockService := mocks.NewCEPService(t)
	ctrl := NewCEPController(mockService)

	expectedData := json.RawMessage(`{"cep":"01001-000","logradouro":"Praça da Sé","localidade":"São Paulo"}`)
	mockService.On("LookupCEP", mock.Anything, "01001000").Return(&model.CEPResult{
		Provider: "ViaCEP",
		Data:     expectedData,
	}, nil)

	r := setupRouter(ctrl)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/cep/01001000", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.CEPResult
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ViaCEP", response.Provider)
}

func TestLookupCEP_Timeout(t *testing.T) {
	mockService := mocks.NewCEPService(t)
	ctrl := NewCEPController(mockService)

	mockService.On("LookupCEP", mock.Anything, "99999999").Return(
		(*model.CEPResult)(nil),
		ports.ErrTimeout,
	)

	r := setupRouter(ctrl)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/cep/99999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusGatewayTimeout, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "timeout")
}

func TestLookupCEP_InternalError(t *testing.T) {
	mockService := mocks.NewCEPService(t)
	ctrl := NewCEPController(mockService)

	mockService.On("LookupCEP", mock.Anything, "00000000").Return(
		(*model.CEPResult)(nil),
		ports.ErrBothFailed,
	)

	r := setupRouter(ctrl)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/cep/00000000", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "erro interno ao consultar CEP", response["error"])
}

func TestLookupCEP_InvalidCEP(t *testing.T) {
	cases := []string{"abc", "1234567", "123456789", "1234-567"}
	for _, cep := range cases {
		mockService := mocks.NewCEPService(t)
		ctrl := NewCEPController(mockService)

		r := setupRouter(ctrl)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/cep/"+cep, nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "CEP: %q", cep)
		mockService.AssertNotCalled(t, "LookupCEP")
	}
}
