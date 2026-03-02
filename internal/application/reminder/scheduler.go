package reminder

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

const jobTimeout = 2 * time.Hour

type Scheduler struct {
	cron       *cron.Cron
	logger     *slog.Logger
	dispatcher *Dispatcher
}

func NewScheduler(logger *slog.Logger, sendTime string, dispatcher *Dispatcher) (*Scheduler, error) {
	hour, minute, err := parseClock(sendTime)
	if err != nil {
		return nil, err
	}

	spec := fmt.Sprintf("%d %d * * *", minute, hour)
	engine := cron.New(cron.WithLocation(time.Local))

	s := &Scheduler{
		cron:       engine,
		logger:     logger,
		dispatcher: dispatcher,
	}

	if _, err := engine.AddFunc(spec, s.runJob); err != nil {
		return nil, fmt.Errorf("register cron job: %w", err)
	}

	return s, nil
}

func (s *Scheduler) Start() {
	s.cron.Start()
	s.logger.Info("daily reminder scheduler started")
}

func (s *Scheduler) Shutdown(ctx context.Context) error {
	doneCtx := s.cron.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-doneCtx.Done():
		return nil
	}
}

func (s *Scheduler) runJob() {
	jobID := generateJobID()
	s.logger.Info("daily reminder dispatch started", "job_id", jobID)

	ctx, cancel := context.WithTimeout(withJobID(context.Background(), jobID), jobTimeout)
	defer cancel()

	if err := s.dispatcher.Dispatch(ctx); err != nil {
		s.logger.Error("daily reminder dispatch failed", "job_id", jobID, "error", err)
		return
	}

	s.logger.Info("daily reminder dispatch completed", "job_id", jobID)
}

func parseClock(raw string) (hour int, minute int, err error) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), " ", ""))
	parsed, err := time.Parse("3:04PM", normalized)
	if err != nil {
		return 0, 0, fmt.Errorf("parse email send time: %w", err)
	}

	return parsed.Hour(), parsed.Minute(), nil
}

func generateJobID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}
