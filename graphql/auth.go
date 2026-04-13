package graphql

import (
	"net/http"
	"strings"
)

// AuthMiddlewareOption configures the auth middleware
type AuthMiddlewareOption func(*authMiddlewareConfig)

type authMiddlewareConfig struct {
	allowUnauthIntrospection bool
	skipPaths                []string
}

// WithAllowUnauthIntrospection allows introspection queries without authentication
func WithAllowUnauthIntrospection(allow bool) AuthMiddlewareOption {
	return func(c *authMiddlewareConfig) {
		c.allowUnauthIntrospection = allow
	}
}

// WithSkipPaths defines paths that skip authentication. Supports wildcard suffix, e.g. "/webhooks/*".
func WithSkipPaths(paths ...string) AuthMiddlewareOption {
	return func(c *authMiddlewareConfig) {
		c.skipPaths = append(c.skipPaths, paths...)
	}
}

// matchesSkipPath проверяет, совпадает ли путь с паттерном.
// Если паттерн оканчивается на "/*" — проверяет префикс.
func matchesSkipPath(pattern, urlPath string) bool {
	if strings.HasSuffix(pattern, "/*") {
		return strings.HasPrefix(urlPath, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == urlPath
}

// AuthMiddlewareFunc is the function that validates the auth header and returns user ID
type AuthMiddlewareFunc func(token string) (any, error)

// AuthMiddleware creates a GraphQL authentication middleware
func AuthMiddleware(authFunc AuthMiddlewareFunc, opts ...AuthMiddlewareOption) func(http.Handler) http.Handler {
	config := &authMiddlewareConfig{
		allowUnauthIntrospection: false,
		skipPaths:                []string{"/health", "/", "/ready"},
	}

	for _, opt := range opts {
		opt(config)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for specified paths
			for _, pattern := range config.skipPaths {
				if matchesSkipPath(pattern, r.URL.Path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Skip authentication for introspection queries if allowed
			if config.allowUnauthIntrospection && IsIntrospectionRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract and validate token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "authorization header required", http.StatusUnauthorized)
				return
			}

			// Remove "Bearer " prefix
			token := authHeader
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}

			// Validate token using provided auth function
			_, err := authFunc(token)
			if err != nil {
				http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
