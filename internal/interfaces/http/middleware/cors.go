package middleware

import (
	"net/http"
	"strings"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

func CORS(cfg CORSConfig, next http.Handler) http.Handler {
	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		allowOrigin := resolveAllowedOrigin(origin, cfg.AllowedOrigins)
		if allowOrigin == "" {
			if r.Method == http.MethodOptions {
				http.Error(w, "cors origin not allowed", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if allowOrigin == "*" && cfg.AllowCredentials {
			allowOrigin = origin
		}

		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		appendVaryHeader(w.Header(), "Origin")
		w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		if cfg.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func appendVaryHeader(header http.Header, value string) {
	existing := strings.TrimSpace(header.Get("Vary"))
	if existing == "" {
		header.Set("Vary", value)
		return
	}
	if strings.Contains(existing, value) {
		return
	}
	header.Set("Vary", existing+", "+value)
}

func resolveAllowedOrigin(origin string, allowed []string) string {
	for _, candidate := range allowed {
		value := strings.TrimSpace(candidate)
		if value == "" {
			continue
		}
		if value == "*" {
			return "*"
		}
		if strings.EqualFold(value, origin) {
			return origin
		}
	}
	return ""
}
