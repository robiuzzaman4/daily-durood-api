package reminder

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
)

type Dispatcher struct {
	users       user.Repository
	emailSender *EmailService
	workerLimit int
	logger      *slog.Logger
}

func NewDispatcher(users user.Repository, emailSender *EmailService, workerLimit int, logger *slog.Logger) *Dispatcher {
	if workerLimit <= 0 {
		workerLimit = 1
	}

	return &Dispatcher{
		users:       users,
		emailSender: emailSender,
		workerLimit: workerLimit,
		logger:      logger,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context) error {
	recipients, err := d.users.ListSubscribed(ctx)
	if err != nil {
		return fmt.Errorf("load subscribed users: %w", err)
	}
	if len(recipients) == 0 {
		d.logger.Info("no subscribed users found for daily reminder")
		return nil
	}

	jobs := make(chan user.User)
	workers := d.workerLimit
	if len(recipients) < workers {
		workers = len(recipients)
	}

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for recipient := range jobs {
				if err := d.emailSender.SendDailyDurood(ctx, recipient); err != nil {
					d.logger.Error("failed to send daily reminder", "user_id", recipient.ID, "email", recipient.Email, "error", err)
					continue
				}

				if err := d.users.IncrementTotalEmailReceived(ctx, recipient.ID, 1); err != nil {
					d.logger.Error("failed to increment total_email_received", "user_id", recipient.ID, "error", err)
					continue
				}

				d.logger.Info("daily reminder sent", "user_id", recipient.ID, "email", recipient.Email)
			}
		}()
	}

	for _, recipient := range recipients {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		case jobs <- recipient:
		}
	}

	close(jobs)
	wg.Wait()

	return nil
}
