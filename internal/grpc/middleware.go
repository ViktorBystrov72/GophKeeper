// Package grpc содержит gRPC сервер и обработчики для GophKeeper.
package grpc

import (
	"context"
	"strings"

	"github.com/GophKeeper/internal/auth"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor создает gRPC interceptor для проверки JWT токенов.
func AuthInterceptor(authService *auth.Service, logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Пропускаем аутентификацию для публичных методов
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Извлекаем токен из метаданных
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn("No metadata in request")
			return nil, status.Error(codes.Unauthenticated, "user not authenticated")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			logger.Warn("No authorization header")
			return nil, status.Error(codes.Unauthenticated, "user not authenticated")
		}

		authHeader := authHeaders[0]

		if !strings.HasPrefix(authHeader, "Bearer ") {
			logger.Warn("Invalid authorization header format")
			return nil, status.Error(codes.Unauthenticated, "user not authenticated")
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Валидируем токен
		claims, err := authService.ValidateToken(token)
		if err != nil {
			logger.Warn("Invalid token", zap.Error(err))
			return nil, status.Error(codes.Unauthenticated, "user not authenticated")
		}

		// Добавляем информацию о пользователе в контекст
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)

		return handler(ctx, req)
	}
}

// isPublicMethod проверяет, является ли метод публичным (не требует аутентификации).
func isPublicMethod(method string) bool {
	publicMethods := []string{
		"/gophkeeper.GophKeeper/Register",
		"/gophkeeper.GophKeeper/Login",
		"/gophkeeper.GophKeeper/RefreshToken",
		"/gophkeeper.GophKeeper/GenerateOTP",
		"/gophkeeper.GophKeeper/CreateOTPSecret",
	}

	for _, publicMethod := range publicMethods {
		if method == publicMethod {
			return true
		}
	}
	return false
}
