package user

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("user not found")

type Repository interface {
	Create(ctx context.Context, u User) (*User, error)
	Update(ctx context.Context, u User) (*User, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListSubscribed(ctx context.Context) ([]User, error)
	ListByScope(ctx context.Context, requesterID string, requesterRole Role) ([]User, error)
	IncrementTotalEmailReceived(ctx context.Context, id string, delta int) error
	CountUsers(ctx context.Context) (int64, error)
	CountTotalEmailsSent(ctx context.Context) (int64, error)
}
