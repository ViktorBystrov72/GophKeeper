// Package grpc содержит gRPC сервер и обработчики для GophKeeper.
package grpc

// Типы для ключей контекста
type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UsernameKey contextKey = "username"
) 