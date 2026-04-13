package jwt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	pkgerrors "github.com/pkg/errors"
)

// contextKey — приватный тип ключей контекста для предотвращения коллизий между пакетами.
type contextKey string

const (
	// UserKey — ключ для хранения [Claims] в контексте запроса.
	UserKey contextKey = "user"
	// UserIDKey — ключ для хранения числового идентификатора пользователя в контексте запроса.
	UserIDKey contextKey = "id"
	// AccessTokenKey — ключ для хранения строки токена в контексте запроса.
	AccessTokenKey contextKey = "accessToken"
)

const defaultTokenTTL = 90 * 24 * time.Hour

// Config хранит параметры для создания [Service].
type Config struct {
	// Secret — секрет для подписи и валидации токенов HS256.
	Secret string
	// TokenTTL — время жизни токена. Если не задано — используется 90 дней.
	TokenTTL time.Duration
}

// Claims — данные, закодированные в JWT-токене.
type Claims struct {
	// UserID — идентификатор пользователя.
	UserID int `json:"id"`
	gojwt.RegisteredClaims
}

// Service реализует JWT-аутентификацию: создание и валидацию токенов,
// а также работу с контекстом запроса.
type Service struct {
	cfg Config
}

// New создаёт новый [Service] с заданной конфигурацией.
func New(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// GenerateToken создаёт подписанный JWT-токен для заданного пользователя.
// Возвращает ошибку, если userID <= 0 или подпись не удалась.
func (s *Service) GenerateToken(userID int) (string, error) {
	if userID <= 0 {
		return "", errors.New("invalid user id")
	}

	ttl := s.cfg.TokenTTL
	if ttl == 0 {
		ttl = defaultTokenTTL
	}

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  gojwt.NewNumericDate(time.Now()),
		},
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return "", pkgerrors.Wrap(err, "failed to sign token")
	}

	return signed, nil
}

// VerifyToken валидирует JWT-токен и возвращает его данные.
// Проверяет алгоритм подписи, подпись, срок действия и корректность userID.
func (s *Service) VerifyToken(tokenString string) (*Claims, error) {
	token, err := gojwt.ParseWithClaims(tokenString, &Claims{}, func(token *gojwt.Token) (any, error) {
		if _, ok := token.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to parse token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.UserID <= 0 {
		return nil, errors.New("invalid user id in token")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	return claims, nil
}

// GetClaimsFromContext извлекает [Claims] из контекста запроса.
// Возвращает ошибку, если claims отсутствуют или имеют неверный тип.
func GetClaimsFromContext(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(UserKey).(*Claims)
	if !ok || claims == nil {
		return nil, errors.New("unauthorized")
	}
	return claims, nil
}

// GetAccessTokenFromContext извлекает строку токена из контекста запроса.
// Возвращает пустую строку, если токен отсутствует.
func GetAccessTokenFromContext(ctx context.Context) string {
	token, ok := ctx.Value(AccessTokenKey).(string)
	if !ok {
		return ""
	}
	return token
}

// GetUserID извлекает идентификатор пользователя из контекста запроса.
// Возвращает ошибку, если пользователь не аутентифицирован.
func GetUserID(ctx context.Context) (int, error) {
	id, ok := ctx.Value(UserIDKey).(int)
	if !ok {
		return 0, errors.New("user not authenticated")
	}
	return id, nil
}

// GetAccessToken извлекает строку токена из контекста запроса.
// Возвращает ошибку, если токен отсутствует.
func GetAccessToken(ctx context.Context) (string, error) {
	token := GetAccessTokenFromContext(ctx)
	if token == "" {
		return "", errors.New("access token not found in context")
	}
	return token, nil
}

// IsAuthenticated проверяет, аутентифицирован ли пользователь в контексте.
func IsAuthenticated(ctx context.Context) bool {
	_, err := GetUserID(ctx)
	return err == nil
}

// ExtractTokenFromBearer извлекает токен из заголовка Authorization
// в формате "Bearer {token}". Регистр слова Bearer не важен.
func ExtractTokenFromBearer(bearer string) (string, error) {
	if bearer == "" {
		return "", errors.New("authorization header required")
	}

	parts := strings.SplitN(bearer, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization header format")
	}

	if parts[1] == "" {
		return "", errors.New("empty token")
	}

	return parts[1], nil
}
