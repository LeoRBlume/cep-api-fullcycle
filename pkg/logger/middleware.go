package logger

import (
	"net/http"

	"github.com/google/uuid"
)

// TraceMiddleware injeta um trace_id no contexto de cada requisição HTTP.
// Lê X-Correlation-ID do header de entrada; gera um UUID v4 se ausente.
// Devolve o mesmo valor no header X-Correlation-ID da response.
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Correlation-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}
		w.Header().Set("X-Correlation-ID", traceID)
		ctx := WithTraceID(r.Context(), traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
