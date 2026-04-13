// Package jwt предоставляет переиспользуемую реализацию JWT-аутентификации
// на основе алгоритма HS256.
//
// Пакет включает:
//   - [Service] — создание и валидация токенов, работа с контекстом запроса
//   - [NewMiddleware] — удобный конструктор HTTP-мидлвара для быстрого подключения
//   - [Service.Middleware] — HTTP-мидлвар для автоматической аутентификации входящих запросов
//
// Использование:
//
//	// Быстрое подключение мидлвара через NewMiddleware
//	mux.Use(jwt.NewMiddleware(cfg.JWTSecret, cfg.JWTExpiration,
//	    jwt.WithSkipPaths("/live", "/webhooks/*"),
//	))
//
//	// Или расширенный вариант через Service
//	svc := jwt.New(jwt.Config{Secret: "my-secret"})
//	mux.Use(svc.Middleware(
//	    jwt.WithAllowUnauthIntrospection(true),
//	    jwt.WithSkipPaths("/metrics"),
//	))
//
//	// Создание токена
//	token, err := svc.GenerateToken(userID)
//
//	// Валидация токена
//	claims, err := svc.VerifyToken(token)
//
//	// Извлечение данных пользователя из контекста (с ошибкой)
//	userID, err := jwt.GetUserID(ctx)
//	token, err := jwt.GetAccessToken(ctx)
//	ok := jwt.IsAuthenticated(ctx)
//
//	// Низкоуровневые хелперы
//	claims, err := jwt.GetClaimsFromContext(ctx)
//	token := jwt.GetAccessTokenFromContext(ctx)
//
// Конфигурация:
//
//	Config.Secret   — секрет для подписи токенов HS256 (обязательно)
//	Config.TokenTTL — время жизни токена (default: 90 дней)
//
// Ограничения:
//
//   - Потокобезопасность: да
//   - Алгоритм подписи: HS256 (HMAC-SHA256)
//   - Config.Secret не может быть пустым
package jwt
