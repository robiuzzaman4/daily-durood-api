package reminder

import "context"

type contextKey string

const jobIDContextKey contextKey = "job_id"

func withJobID(ctx context.Context, jobID string) context.Context {
	return context.WithValue(ctx, jobIDContextKey, jobID)
}

func jobIDFromContext(ctx context.Context) string {
	value, ok := ctx.Value(jobIDContextKey).(string)
	if !ok {
		return ""
	}
	return value
}
