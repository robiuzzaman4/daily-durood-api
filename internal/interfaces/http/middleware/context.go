package middleware

import (
	"context"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
)

type contextKey string

const (
	userContextKey      contextKey = "request_user"
	requestIDContextKey contextKey = "request_id"
)

func WithUser(ctx context.Context, u *user.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

func UserFromContext(ctx context.Context) (*user.User, bool) {
	u, ok := ctx.Value(userContextKey).(*user.User)
	if !ok || u == nil {
		return nil, false
	}
	return u, true
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDContextKey).(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}
