package jwt_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/platform/jwt"
)

// generateToken генерирует валидный JWT-токен для тестов.
func generateToken(t *testing.T, userID int, secret string) string {
	t.Helper()
	svc := jwt.New(jwt.Config{Secret: secret})
	token, err := svc.GenerateToken(userID)
	require.NoError(t, err)
	return token
}

// обработчик-заглушка, фиксирующий вызов и код ответа.
func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	svc := jwt.New(jwt.Config{Secret: "test-secret"})

	t.Run("valid_token_passes_through", func(t *testing.T) {
		t.Parallel()

		// Arrange
		token, err := svc.GenerateToken(99)
		require.NoError(t, err)

		called := false
		handler := svc.Middleware()(okHandler(&called))

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("valid_token_sets_context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		token, err := svc.GenerateToken(55)
		require.NoError(t, err)

		var gotUserID int
		checkCtxHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, e := jwt.GetClaimsFromContext(r.Context())
			require.NoError(t, e)
			gotUserID = claims.UserID
			w.WriteHeader(http.StatusOK)
		})
		handler := svc.Middleware()(checkCtxHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.Equal(t, 55, gotUserID)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing_auth_header_returns_401", func(t *testing.T) {
		t.Parallel()

		// Arrange
		called := false
		handler := svc.Middleware()(okHandler(&called))

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.False(t, called)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid_token_returns_401", func(t *testing.T) {
		t.Parallel()

		// Arrange
		called := false
		handler := svc.Middleware()(okHandler(&called))

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.False(t, called)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("skip_path_health_passes_without_auth", func(t *testing.T) {
		t.Parallel()

		// Arrange
		called := false
		handler := svc.Middleware()(okHandler(&called))

		req := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("custom_skip_path_passes_without_auth", func(t *testing.T) {
		t.Parallel()

		// Arrange
		called := false
		handler := svc.Middleware(
			jwt.WithSkipPaths("/metrics"),
		)(okHandler(&called))

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("prefix_pattern_skips_any_subpath", func(t *testing.T) {
		t.Parallel()

		// Arrange
		called := false
		handler := svc.Middleware(
			jwt.WithSkipPaths("/webhooks/*"),
		)(okHandler(&called))

		req := httptest.NewRequest(http.MethodPost, "/webhooks/livekit", nil)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("prefix_pattern_does_not_skip_sibling_path", func(t *testing.T) {
		t.Parallel()

		// Arrange
		called := false
		handler := svc.Middleware(
			jwt.WithSkipPaths("/webhooks/*"),
		)(okHandler(&called))

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.False(t, called)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("wrong_secret_returns_401", func(t *testing.T) {
		t.Parallel()

		// Arrange
		otherSvc := jwt.New(jwt.Config{Secret: "other-secret"})
		token, err := otherSvc.GenerateToken(1)
		require.NoError(t, err)

		called := false
		handler := svc.Middleware()(okHandler(&called))

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.False(t, called)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestNewMiddleware(t *testing.T) {
	t.Parallel()

	const secret = "test-secret"
	const ttlDays = 90

	t.Run("valid_token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		token := generateToken(t, 123, secret)
		mw := jwt.NewMiddleware(secret, ttlDays, jwt.WithSkipPaths("/live", "/webhooks/*"))
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := jwt.GetUserID(r.Context())
			assert.NoError(t, err)
			assert.Equal(t, 123, userID)

			accessToken, err := jwt.GetAccessToken(r.Context())
			assert.NoError(t, err)
			assert.Equal(t, token, accessToken)

			assert.True(t, jwt.IsAuthenticated(r.Context()))
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing_auth_header", func(t *testing.T) {
		t.Parallel()

		// Arrange
		called := false
		mw := jwt.NewMiddleware(secret, ttlDays)
		handler := mw(okHandler(&called))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.False(t, called)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid_auth_format", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name   string
			header string
		}{
			{"no_bearer_prefix", "invalid-token"},
			{"only_bearer", "Bearer"},
			{"wrong_prefix", "Basic token"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				called := false
				mw := jwt.NewMiddleware(secret, ttlDays)
				handler := mw(okHandler(&called))

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", tc.header)
				rec := httptest.NewRecorder()

				// Act
				handler.ServeHTTP(rec, req)

				// Assert
				assert.False(t, called)
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			})
		}
	})

	t.Run("invalid_token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		called := false
		mw := jwt.NewMiddleware(secret, ttlDays)
		handler := mw(okHandler(&called))

		testCases := []struct {
			name  string
			token string
		}{
			{"invalid_signature", "Bearer invalid.token.here"},
			{"malformed_token", "Bearer not-a-jwt"},
			{"empty_token", "Bearer "},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", tc.token)
				rec := httptest.NewRecorder()

				handler.ServeHTTP(rec, req)

				assert.False(t, called)
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			})
		}
	})

	t.Run("wrong_signing_method", func(t *testing.T) {
		t.Parallel()

		// Arrange
		token := gojwt.NewWithClaims(gojwt.SigningMethodNone, gojwt.MapClaims{"id": 123})
		tokenString, _ := token.SignedString(gojwt.UnsafeAllowNoneSignatureType)

		called := false
		mw := jwt.NewMiddleware(secret, ttlDays)
		handler := mw(okHandler(&called))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.False(t, called)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("expired_token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		claims := &jwt.Claims{
			UserID: 123,
			RegisteredClaims: gojwt.RegisteredClaims{
				ExpiresAt: gojwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
				IssuedAt:  gojwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}
		raw := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
		tokenString, err := raw.SignedString([]byte(secret))
		require.NoError(t, err)

		called := false
		mw := jwt.NewMiddleware(secret, ttlDays)
		handler := mw(okHandler(&called))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.False(t, called)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("skip_paths", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mw := jwt.NewMiddleware(secret, ttlDays, jwt.WithSkipPaths("/live", "/webhooks/*"))

		for _, path := range []string{"/live", "/webhooks/livekit", "/webhooks/some/nested"} {
			t.Run(path, func(t *testing.T) {
				t.Parallel()

				called := false
				handler := mw(okHandler(&called))

				req := httptest.NewRequest(http.MethodGet, path, nil)
				rec := httptest.NewRecorder()

				handler.ServeHTTP(rec, req)

				assert.True(t, called)
				assert.Equal(t, http.StatusOK, rec.Code)
			})
		}
	})
}
