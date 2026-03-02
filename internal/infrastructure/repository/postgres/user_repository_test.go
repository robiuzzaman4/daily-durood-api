package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
)

func TestIncrementTotalEmailReceived_AtomicUpdate(t *testing.T) {
	var capturedSQL string
	var capturedArgs []any

	db := &fakeDB{
		execFn: func(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			capturedSQL = sql
			capturedArgs = args
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}

	repo := NewUserRepositoryWithQuerier(db)
	err := repo.IncrementTotalEmailReceived(context.Background(), "user-1", 1)
	if err != nil {
		t.Fatalf("IncrementTotalEmailReceived() returned error: %v", err)
	}

	if !strings.Contains(capturedSQL, "total_email_received = total_email_received + $2") {
		t.Fatalf("expected atomic increment SQL, got: %s", capturedSQL)
	}
	if len(capturedArgs) != 2 {
		t.Fatalf("expected 2 args, got %d", len(capturedArgs))
	}
	if capturedArgs[0] != "user-1" || capturedArgs[1] != 1 {
		t.Fatalf("unexpected args: %#v", capturedArgs)
	}
}

func TestIncrementTotalEmailReceived_NotFound(t *testing.T) {
	db := &fakeDB{
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	}

	repo := NewUserRepositoryWithQuerier(db)
	err := repo.IncrementTotalEmailReceived(context.Background(), "missing-user", 1)
	if !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("expected user.ErrNotFound, got: %v", err)
	}
}

func TestCountTotalEmailsSent(t *testing.T) {
	db := &fakeDB{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{
				scanFn: func(dest ...any) error {
					total := dest[0].(*int64)
					*total = 42
					return nil
				},
			}
		},
	}

	repo := NewUserRepositoryWithQuerier(db)
	total, err := repo.CountTotalEmailsSent(context.Background())
	if err != nil {
		t.Fatalf("CountTotalEmailsSent() returned error: %v", err)
	}
	if total != 42 {
		t.Fatalf("expected total=42, got %d", total)
	}
}

type fakeDB struct {
	execFn     func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	queryFn    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (f *fakeDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if f.execFn == nil {
		return pgconn.CommandTag{}, nil
	}
	return f.execFn(ctx, sql, arguments...)
}

func (f *fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if f.queryFn == nil {
		return nil, nil
	}
	return f.queryFn(ctx, sql, args...)
}

func (f *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if f.queryRowFn == nil {
		return fakeRow{scanFn: func(_ ...any) error { return nil }}
	}
	return f.queryRowFn(ctx, sql, args...)
}

type fakeRow struct {
	scanFn func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	return r.scanFn(dest...)
}
