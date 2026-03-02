package reminder

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
)

func TestEmailService_RetriesAndSucceeds(t *testing.T) {
	client := &fakeEmailClient{
		failuresByRecipient: map[string]int{
			"user@example.com": 2,
		},
	}

	service := NewEmailService(client)
	service.retryDelay = 1 * time.Millisecond
	service.maxRetries = 3

	err := service.SendDailyDurood(context.Background(), user.User{
		ID:    "u1",
		Name:  "User One",
		Email: "user@example.com",
	})
	if err != nil {
		t.Fatalf("SendDailyDurood() returned error: %v", err)
	}

	if got := client.calls["user@example.com"]; got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestDispatcher_IncrementsOnlyOnSuccessfulSend(t *testing.T) {
	repo := &fakeUserRepository{
		subscribed: []user.User{
			{ID: "u1", Name: "One", Email: "one@example.com", IsSubscribed: true},
			{ID: "u2", Name: "Two", Email: "two@example.com", IsSubscribed: true},
			{ID: "u3", Name: "Three", Email: "three@example.com", IsSubscribed: true},
		},
		increments: map[string]int{},
	}

	client := &fakeEmailClient{
		failuresByRecipient: map[string]int{
			"two@example.com": 10, // always fails with maxRetries=3
		},
	}
	service := NewEmailService(client)
	service.retryDelay = 1 * time.Millisecond
	service.maxRetries = 3

	dispatcher := NewDispatcher(repo, service, 2, testLogger())
	err := dispatcher.Dispatch(withJobID(context.Background(), "job-test"))
	if err != nil {
		t.Fatalf("Dispatch() returned error: %v", err)
	}

	if repo.increments["u1"] != 1 {
		t.Fatalf("expected u1 increment=1, got %d", repo.increments["u1"])
	}
	if repo.increments["u3"] != 1 {
		t.Fatalf("expected u3 increment=1, got %d", repo.increments["u3"])
	}
	if repo.increments["u2"] != 0 {
		t.Fatalf("expected u2 increment=0, got %d", repo.increments["u2"])
	}
}

type fakeEmailClient struct {
	mu                  sync.Mutex
	failuresByRecipient map[string]int
	calls               map[string]int
}

func (f *fakeEmailClient) Send(_ context.Context, email OutboundEmail) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls[email.To]++

	if f.failuresByRecipient[email.To] > 0 {
		f.failuresByRecipient[email.To]--
		return errors.New("send failed")
	}

	return nil
}

type fakeUserRepository struct {
	mu         sync.Mutex
	subscribed []user.User
	increments map[string]int
}

func (f *fakeUserRepository) Create(context.Context, user.User) (*user.User, error) { return nil, nil }
func (f *fakeUserRepository) Update(context.Context, user.User) (*user.User, error) { return nil, nil }
func (f *fakeUserRepository) Delete(context.Context, string) error                  { return nil }
func (f *fakeUserRepository) GetByID(context.Context, string) (*user.User, error)   { return nil, nil }
func (f *fakeUserRepository) GetByEmail(context.Context, string) (*user.User, error) {
	return nil, nil
}
func (f *fakeUserRepository) ListSubscribed(context.Context) ([]user.User, error) {
	return f.subscribed, nil
}
func (f *fakeUserRepository) ListByScope(context.Context, string, user.Role) ([]user.User, error) {
	return nil, nil
}
func (f *fakeUserRepository) IncrementTotalEmailReceived(_ context.Context, id string, delta int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.increments[id] += delta
	return nil
}
func (f *fakeUserRepository) CountUsers(context.Context) (int64, error)           { return 0, nil }
func (f *fakeUserRepository) CountTotalEmailsSent(context.Context) (int64, error) { return 0, nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
