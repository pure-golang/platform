package jwt_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/platform/jwt"
)

func TestGenerateToken(t *testing.T) {
	t.Parallel()

	svc := jwt.New(jwt.Config{Secret: "test-secret"})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		// Act
		token, err := svc.GenerateToken(42)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("invalid_user_id_zero", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := svc.GenerateToken(0)

		// Assert
		assert.Error(t, err)
	})

	t.Run("invalid_user_id_negative", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := svc.GenerateToken(-1)

		// Assert
		assert.Error(t, err)
	})

	t.Run("custom_ttl", func(t *testing.T) {
		t.Parallel()

		// Arrange
		svcWithTTL := jwt.New(jwt.Config{
			Secret:   "test-secret",
			TokenTTL: time.Hour,
		})

		// Act
		token, err := svcWithTTL.GenerateToken(1)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func TestVerifyToken(t *testing.T) {
	t.Parallel()

	svc := jwt.New(jwt.Config{Secret: "test-secret"})

	t.Run("valid_token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		token, err := svc.GenerateToken(42)
		require.NoError(t, err)

		// Act
		claims, err := svc.VerifyToken(token)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 42, claims.UserID)
	})

	t.Run("wrong_secret", func(t *testing.T) {
		t.Parallel()

		// Arrange
		otherSvc := jwt.New(jwt.Config{Secret: "other-secret"})
		token, err := otherSvc.GenerateToken(1)
		require.NoError(t, err)

		// Act
		_, err = svc.VerifyToken(token)

		// Assert
		assert.Error(t, err)
	})

	t.Run("empty_token", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := svc.VerifyToken("")

		// Assert
		assert.Error(t, err)
	})

	t.Run("malformed_token", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := svc.VerifyToken("not.a.jwt")

		// Assert
		assert.Error(t, err)
	})
}

func TestExtractTokenFromBearer(t *testing.T) {
	t.Parallel()

	t.Run("valid_bearer", func(t *testing.T) {
		t.Parallel()

		// Act
		token, err := jwt.ExtractTokenFromBearer("Bearer mytoken123")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "mytoken123", token)
	})

	t.Run("case_insensitive_bearer", func(t *testing.T) {
		t.Parallel()

		// Act
		token, err := jwt.ExtractTokenFromBearer("bearer mytoken123")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "mytoken123", token)
	})

	t.Run("empty_header", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := jwt.ExtractTokenFromBearer("")

		// Assert
		assert.Error(t, err)
	})

	t.Run("missing_bearer_prefix", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := jwt.ExtractTokenFromBearer("mytoken123")

		// Assert
		assert.Error(t, err)
	})

	t.Run("empty_token_after_bearer", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := jwt.ExtractTokenFromBearer("Bearer ")

		// Assert
		assert.Error(t, err)
	})
}

func TestGetClaimsFromContext(t *testing.T) {
	t.Parallel()

	svc := jwt.New(jwt.Config{Secret: "test-secret"})

	t.Run("claims_present", func(t *testing.T) {
		t.Parallel()

		// Arrange
		token, err := svc.GenerateToken(7)
		require.NoError(t, err)
		claims, err := svc.VerifyToken(token)
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), jwt.UserKey, claims)

		// Act
		got, err := jwt.GetClaimsFromContext(ctx)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 7, got.UserID)
	})

	t.Run("no_claims_in_context", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := jwt.GetClaimsFromContext(context.Background())

		// Assert
		assert.Error(t, err)
	})
}

func TestGetUserID(t *testing.T) {
	t.Parallel()

	t.Run("not_authenticated", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := jwt.GetUserID(context.Background())

		// Assert
		assert.Error(t, err)
	})

	t.Run("wrong_type", func(t *testing.T) {
		t.Parallel()

		// Arrange
		ctx := context.WithValue(context.Background(), jwt.UserIDKey, "not-an-int")

		// Act
		_, err := jwt.GetUserID(ctx)

		// Assert
		assert.Error(t, err)
	})

	t.Run("authenticated", func(t *testing.T) {
		t.Parallel()

		// Arrange
		ctx := context.WithValue(context.Background(), jwt.UserIDKey, int(42))

		// Act
		id, err := jwt.GetUserID(ctx)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 42, id)
	})
}

func TestGetAccessToken(t *testing.T) {
	t.Parallel()

	t.Run("not_authenticated", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := jwt.GetAccessToken(context.Background())

		// Assert
		assert.Error(t, err)
	})

	t.Run("present", func(t *testing.T) {
		t.Parallel()

		// Arrange
		ctx := context.WithValue(context.Background(), jwt.AccessTokenKey, "mytoken")

		// Act
		token, err := jwt.GetAccessToken(ctx)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "mytoken", token)
	})
}

func TestIsAuthenticated(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		ctx      context.Context
		expected bool
	}{
		{
			name:     "no_context_value",
			ctx:      context.Background(),
			expected: false,
		},
		{
			name:     "wrong_type",
			ctx:      context.WithValue(context.Background(), jwt.UserIDKey, "string"),
			expected: false,
		},
		{
			name:     "authenticated",
			ctx:      context.WithValue(context.Background(), jwt.UserIDKey, int(123)),
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := jwt.IsAuthenticated(tc.ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetAccessTokenFromContext(t *testing.T) {
	t.Parallel()

	t.Run("token_present", func(t *testing.T) {
		t.Parallel()

		// Arrange
		ctx := context.WithValue(context.Background(), jwt.AccessTokenKey, "mytoken")

		// Act
		got := jwt.GetAccessTokenFromContext(ctx)

		// Assert
		assert.Equal(t, "mytoken", got)
	})

	t.Run("no_token_in_context", func(t *testing.T) {
		t.Parallel()

		// Act
		got := jwt.GetAccessTokenFromContext(context.Background())

		// Assert
		assert.Empty(t, got)
	})
}
