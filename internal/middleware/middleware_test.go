package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GophKeeper/internal/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAuthMiddleware(t *testing.T) {
	authService := auth.NewService("test-secret")
	logger, _ := zap.NewDevelopment()

	// Создаем тестовый пользователь
	userID := uuid.New()
	username := "testuser"
	token, _, err := authService.GenerateToken(userID, username)
	require.NoError(t, err)

	// Создаем middleware
	middleware := AuthMiddleware(authService, logger)

	// Создаем тестовый handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDFromCtx, ok := GetUserIDFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, userID, userIDFromCtx)
		w.WriteHeader(http.StatusOK)
	})

	// Оборачиваем handler в middleware
	wrappedHandler := middleware(handler)

	t.Run("valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid auth header format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat "+token)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestLoggingMiddleware(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	middleware := LoggingMiddleware(logger)

	// Создаем тестовый handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test response", w.Body.String())
}

func TestCORSMiddleware(t *testing.T) {
	middleware := CORSMiddleware()

	// Создаем тестовый handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	t.Run("preflight request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		require.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
		require.Equal(t, "Accept, Authorization, Content-Type, X-CSRF-Token", w.Header().Get("Access-Control-Allow-Headers"))
	})

	t.Run("regular request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestGetUserIDFromContext(t *testing.T) {
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey{}, userID)

	retrievedUserID, ok := GetUserIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, userID, retrievedUserID)

	// Тест с контекстом без userID
	emptyCtx := context.Background()
	_, ok = GetUserIDFromContext(emptyCtx)
	require.False(t, ok)
}
