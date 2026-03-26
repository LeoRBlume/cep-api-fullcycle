package logger

import "context"

type ctxKey int

const (
	ctxKeyTraceID ctxKey = iota
	ctxKeyUserID
)

// WithTraceID retorna um novo contexto com o trace_id injetado.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyTraceID, id)
}

// WithUserID retorna um novo contexto com o user_id injetado.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, id)
}

func extractFromContext(ctx context.Context) (traceID, userID string) {
	if v, ok := ctx.Value(ctxKeyTraceID).(string); ok {
		traceID = v
	}
	if v, ok := ctx.Value(ctxKeyUserID).(string); ok {
		userID = v
	}
	return
}
