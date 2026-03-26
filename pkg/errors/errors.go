package errors

import (
	stderrors "errors"
	"net/http"
)

// Translator mapeia sentinel errors para HTTP status codes.
// O mapeamento é avaliado em ordem de registro e usa errors.Is,
// portanto suporta wrapping.
type Translator struct {
	entries []entry
}

type entry struct {
	sentinel error
	status   int
}

// NewTranslator cria um Translator vazio.
func NewTranslator() *Translator {
	return &Translator{}
}

// Register adiciona um mapeamento de sentinel error para HTTP status code.
// Retorna o próprio Translator para encadeamento.
func (t *Translator) Register(sentinel error, status int) *Translator {
	t.entries = append(t.entries, entry{sentinel, status})
	return t
}

// Translate retorna o HTTP status code para o erro fornecido.
// Usa errors.Is para suportar erros wrapped.
// Retorna 500 se nenhum mapeamento for encontrado ou se err for nil.
func (t *Translator) Translate(err error) int {
	for _, e := range t.entries {
		if stderrors.Is(err, e.sentinel) {
			return e.status
		}
	}
	return http.StatusInternalServerError
}
