package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
)

type UserRepository struct {
	db dbQuerier
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return NewUserRepositoryWithQuerier(db)
}

func NewUserRepositoryWithQuerier(db dbQuerier) *UserRepository {
	return &UserRepository{db: db}
}

type dbQuerier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *UserRepository) Create(ctx context.Context, u user.User) (*user.User, error) {
	const query = `
		INSERT INTO users (id, name, email, is_subscribed, total_email_received, role)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, email, is_subscribed, total_email_received, role
	`

	created := user.User{}
	err := r.db.QueryRow(ctx, query, u.ID, u.Name, u.Email, u.IsSubscribed, u.TotalEmailReceived, u.Role).
		Scan(&created.ID, &created.Name, &created.Email, &created.IsSubscribed, &created.TotalEmailReceived, &created.Role)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &created, nil
}

func (r *UserRepository) Update(ctx context.Context, u user.User) (*user.User, error) {
	const query = `
		UPDATE users
		SET name = $2,
		    email = $3,
		    is_subscribed = $4,
		    total_email_received = $5,
		    role = $6,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, email, is_subscribed, total_email_received, role
	`

	updated := user.User{}
	err := r.db.QueryRow(ctx, query, u.ID, u.Name, u.Email, u.IsSubscribed, u.TotalEmailReceived, u.Role).
		Scan(&updated.ID, &updated.Name, &updated.Email, &updated.IsSubscribed, &updated.TotalEmailReceived, &updated.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update user %s: %w", u.ID, err)
	}

	return &updated, nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM users WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user %s: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return user.ErrNotFound
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	const query = `
		SELECT id, name, email, is_subscribed, total_email_received, role
		FROM users
		WHERE id = $1
	`

	u := user.User{}
	err := r.db.QueryRow(ctx, query, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.IsSubscribed, &u.TotalEmailReceived, &u.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id %s: %w", id, err)
	}

	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	const query = `
		SELECT id, name, email, is_subscribed, total_email_received, role
		FROM users
		WHERE email = $1
	`

	u := user.User{}
	err := r.db.QueryRow(ctx, query, email).
		Scan(&u.ID, &u.Name, &u.Email, &u.IsSubscribed, &u.TotalEmailReceived, &u.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email %s: %w", email, err)
	}

	return &u, nil
}

func (r *UserRepository) ListSubscribed(ctx context.Context) ([]user.User, error) {
	const query = `
		SELECT id, name, email, is_subscribed, total_email_received, role
		FROM users
		WHERE is_subscribed = TRUE
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list subscribed users: %w", err)
	}
	defer rows.Close()

	return scanUsers(rows)
}

func (r *UserRepository) ListByScope(ctx context.Context, requesterID string, requesterRole user.Role) ([]user.User, error) {
	baseQuery := `
		SELECT id, name, email, is_subscribed, total_email_received, role
		FROM users
	`

	var (
		rows pgx.Rows
		err  error
	)

	if requesterRole == user.RoleAdmin {
		rows, err = r.db.Query(ctx, baseQuery+" ORDER BY created_at ASC")
	} else {
		rows, err = r.db.Query(ctx, baseQuery+" WHERE id = $1 ORDER BY created_at ASC", requesterID)
	}
	if err != nil {
		return nil, fmt.Errorf("list users by scope: %w", err)
	}
	defer rows.Close()

	return scanUsers(rows)
}

func (r *UserRepository) CountUsers(ctx context.Context) (int64, error) {
	const query = `SELECT COUNT(*) FROM users`

	var total int64
	if err := r.db.QueryRow(ctx, query).Scan(&total); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return total, nil
}

func (r *UserRepository) IncrementTotalEmailReceived(ctx context.Context, id string, delta int) error {
	const query = `
		UPDATE users
		SET total_email_received = total_email_received + $2,
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, delta)
	if err != nil {
		return fmt.Errorf("increment total_email_received for user %s: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return user.ErrNotFound
	}

	return nil
}

func (r *UserRepository) CountTotalEmailsSent(ctx context.Context) (int64, error) {
	const query = `SELECT COALESCE(SUM(total_email_received), 0) FROM users`

	var total int64
	if err := r.db.QueryRow(ctx, query).Scan(&total); err != nil {
		return 0, fmt.Errorf("count total emails sent: %w", err)
	}

	return total, nil
}

func scanUsers(rows pgx.Rows) ([]user.User, error) {
	users := make([]user.User, 0)

	for rows.Next() {
		u := user.User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.IsSubscribed, &u.TotalEmailReceived, &u.Role); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users rows: %w", err)
	}

	return users, nil
}
