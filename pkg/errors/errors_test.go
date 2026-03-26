package errors_test

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	apperrors "cep-api/pkg/errors"

	"github.com/stretchr/testify/assert"
)

var (
	errNotFound = stderrors.New("não encontrado")
	errTimeout  = stderrors.New("timeout")
	errFailed   = stderrors.New("falhou")
)

func newTranslator() *apperrors.Translator {
	return apperrors.NewTranslator().
		Register(errNotFound, http.StatusNotFound).
		Register(errTimeout, http.StatusGatewayTimeout).
		Register(errFailed, http.StatusInternalServerError)
}

func TestTranslate_KnownErrors(t *testing.T) {
	tr := newTranslator()

	cases := []struct {
		err      error
		expected int
	}{
		{errNotFound, http.StatusNotFound},
		{errTimeout, http.StatusGatewayTimeout},
		{errFailed, http.StatusInternalServerError},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, tr.Translate(c.err))
	}
}

func TestTranslate_UnknownError_Returns500(t *testing.T) {
	tr := newTranslator()
	assert.Equal(t, http.StatusInternalServerError, tr.Translate(stderrors.New("erro desconhecido")))
}

func TestTranslate_NilError_Returns500(t *testing.T) {
	tr := newTranslator()
	assert.Equal(t, http.StatusInternalServerError, tr.Translate(nil))
}

func TestTranslate_WrappedError(t *testing.T) {
	tr := newTranslator()
	wrapped := fmt.Errorf("contexto adicional: %w", errNotFound)
	assert.Equal(t, http.StatusNotFound, tr.Translate(wrapped))
}

func TestTranslate_EmptyTranslator_Returns500(t *testing.T) {
	tr := apperrors.NewTranslator()
	assert.Equal(t, http.StatusInternalServerError, tr.Translate(errNotFound))
}

func TestRegister_Chaining(t *testing.T) {
	tr := apperrors.NewTranslator().
		Register(errNotFound, http.StatusNotFound).
		Register(errTimeout, http.StatusGatewayTimeout)

	assert.Equal(t, http.StatusNotFound, tr.Translate(errNotFound))
	assert.Equal(t, http.StatusGatewayTimeout, tr.Translate(errTimeout))
}
