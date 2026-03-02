package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
	"github.com/robiuzzaman4/daily-durood-api/internal/infrastructure/auth/clerk"
	"github.com/robiuzzaman4/daily-durood-api/internal/infrastructure/config"
	"github.com/robiuzzaman4/daily-durood-api/internal/interfaces/http/handlers"
	"github.com/robiuzzaman4/daily-durood-api/internal/interfaces/http/middleware"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func NewServer(cfg *config.Config, logger *slog.Logger, db *pgxpool.Pool, users user.Repository) (*Server, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	webhookVerifier, err := clerk.NewWebhookVerifier(cfg.ClerkWebhookSecret)
	if err != nil {
		return nil, fmt.Errorf("create webhook verifier: %w", err)
	}
	mux.Handle("/webhooks/clerk", handlers.NewClerkWebhookHandler(logger, users, webhookVerifier))

	tokenVerifier := clerk.NewTokenVerifier(cfg.ClerkJWKSURL, cfg.ClerkIssuer)
	authMW := middleware.NewAuthMiddleware(clerkTokenVerifierAdapter{verifier: tokenVerifier}, users)

	mux.Handle("/auth/check", authMW.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestUser, ok := middleware.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		_ = writeJSON(w, http.StatusOK, map[string]any{
			"id":    requestUser.ID,
			"email": requestUser.Email,
			"role":  requestUser.Role,
		})
	})))

	mux.Handle("/auth/admin/check", authMW.RequireAuth(middleware.RequireRole(user.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))))

	mux.Handle("/me", authMW.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestUser, ok := middleware.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		profile, err := users.GetByID(r.Context(), requestUser.ID)
		if errors.Is(err, user.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load profile", http.StatusInternalServerError)
			return
		}

		_ = writeJSON(w, http.StatusOK, profile)
	})))

	mux.Handle("/admin/users", authMW.RequireAuth(middleware.RequireRole(user.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestUser, ok := middleware.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		allUsers, err := users.ListByScope(r.Context(), requestUser.ID, requestUser.Role)
		if err != nil {
			http.Error(w, "failed to load users", http.StatusInternalServerError)
			return
		}

		_ = writeJSON(w, http.StatusOK, map[string]any{
			"users": allUsers,
		})
	}))))

	mux.Handle("/metadata", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalUsers, err := users.CountUsers(r.Context())
		if err != nil {
			http.Error(w, "failed to load metadata", http.StatusInternalServerError)
			return
		}

		totalEmailsSent, err := users.CountTotalEmailsSent(r.Context())
		if err != nil {
			http.Error(w, "failed to load metadata", http.StatusInternalServerError)
			return
		}

		_ = writeJSON(w, http.StatusOK, map[string]int64{
			"total_users":       totalUsers,
			"total_emails_sent": totalEmailsSent,
		})
	}))

	return &Server{
		httpServer: &http.Server{
			Addr: fmt.Sprintf(":%s", cfg.ServerPort),
			Handler: middleware.RequireRequestID(
				middleware.LogRequests(
					logger,
					middleware.CORS(middleware.CORSConfig{
						AllowedOrigins:   cfg.CORSAllowedOrigins,
						AllowedMethods:   cfg.CORSAllowedMethods,
						AllowedHeaders:   cfg.CORSAllowedHeaders,
						AllowCredentials: cfg.CORSAllowCredentials,
					}, mux),
				),
			),
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger: logger,
	}, nil
}

func (s *Server) Start() error {
	s.logger.Info("http server started", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

type clerkTokenVerifierAdapter struct {
	verifier *clerk.TokenVerifier
}

func (a clerkTokenVerifierAdapter) Verify(ctx context.Context, token string) (string, error) {
	claims, err := a.verifier.Verify(ctx, token)
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", errors.New("missing token subject")
	}
	return claims.Subject, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}
