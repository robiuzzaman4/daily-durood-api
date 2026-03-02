package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robiuzzaman4/daily-durood-api/internal/domain/user"
)

func TestRequireRole_UnauthorizedWithoutUserContext(t *testing.T) {
	handler := RequireRole(user.RoleAdmin)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireRole_ForbiddenForDisallowedRole(t *testing.T) {
	handler := RequireRole(user.RoleAdmin)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req = req.WithContext(WithUser(req.Context(), &user.User{ID: "u1", Role: user.RoleUser}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestRequireRole_AllowsPermittedRole(t *testing.T) {
	nextCalled := false
	handler := RequireRole(user.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req = req.WithContext(WithUser(req.Context(), &user.User{ID: "a1", Role: user.RoleAdmin}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
