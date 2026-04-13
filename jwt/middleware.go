package jwt

import (
	"context"
	"net/http"
	"strings"
	"time"

	platformgraphql "github.com/pure-golang/platform/graphql"
)

// MiddlewareOption — функция настройки HTTP-мидлвара аутентификации.
type MiddlewareOption func(*middlewareConfig)

type middlewareConfig struct {
	allowUnauthIntrospection bool
	skipPaths                map[string]bool
	skipPrefixes             []string
}

// WithAllowUnauthIntrospection разрешает GraphQL introspection-запросы без аутентификации.
func WithAllowUnauthIntrospection(allow bool) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.allowUnauthIntrospection = allow
	}
}

// WithSkipPaths задаёт дополнительные пути, для которых пропускается аутентификация.
// Пути /healthcheck, / и /ready пропускаются по умолчанию.
//
// Поддерживает два формата:
//   - точный путь: "/metrics" — совпадение строго по r.URL.Path
//   - префиксный паттерн: "/webhooks/*" — совпадение по префиксу "/webhooks/"
func WithSkipPaths(paths ...string) MiddlewareOption {
	return func(c *middlewareConfig) {
		for _, path := range paths {
			if prefix, ok := strings.CutSuffix(path, "*"); ok {
				c.skipPrefixes = append(c.skipPrefixes, prefix)
			} else {
				c.skipPaths[path] = true
			}
		}
	}
}

// NewMiddleware создаёт HTTP-мидлвар JWT-аутентификации.
// ttlDays — срок действия токена в днях (0 — использует дефолт 90 дней).
func NewMiddleware(secret string, ttlDays int, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	svc := New(Config{
		Secret:   secret,
		TokenTTL: time.Duration(ttlDays) * 24 * time.Hour,
	})
	return svc.Middleware(opts...)
}

// Middleware создаёт HTTP-мидлвар JWT-аутентификации.
//
// При успешной валидации токена сохраняет в контексте запроса:
//   - [Claims] по ключу [UserKey]
//   - UserID по ключу [UserIDKey]
//   - строку токена по ключу [AccessTokenKey]
//
// По умолчанию пропускает без проверки: /healthcheck, /, /ready.
func (s *Service) Middleware(opts ...MiddlewareOption) func(http.Handler) http.Handler {
	cfg := &middlewareConfig{
		skipPaths: map[string]bool{
			"/healthcheck": true,
			"/":            true,
			"/ready":       true,
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			for _, prefix := range cfg.skipPrefixes {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			if cfg.allowUnauthIntrospection && platformgraphql.IsIntrospectionRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			tokenString, err := ExtractTokenFromBearer(r.Header.Get("Authorization"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims, err := s.VerifyToken(tokenString)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserKey, claims)
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, AccessTokenKey, tokenString)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
