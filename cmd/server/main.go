// Package main инициализирует и запускает сервер GophKeeper.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/GophKeeper/internal/auth"
	"github.com/GophKeeper/internal/config"
	"github.com/GophKeeper/internal/crypto"
	grpcServer "github.com/GophKeeper/internal/grpc"
	"github.com/GophKeeper/internal/logger"
	"github.com/GophKeeper/internal/middleware"
	"github.com/GophKeeper/internal/otp"
	"github.com/GophKeeper/internal/storage"
	"github.com/GophKeeper/internal/version"
	"go.uber.org/zap"
)

// ServerComponents содержит все компоненты сервера для graceful shutdown
type ServerComponents struct {
	Storage      storage.Storage
	HTTPServer   *http.Server
	GRPCServer   *grpc.Server
	GRPCListener net.Listener
	Logger       *zap.Logger
}

func main() {
	// Создаем контекст с обработкой сигналов
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.LoadServerConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	zapLogger, err := logger.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer zapLogger.Sync()

	version.LogVersionInfo(zapLogger, "GophKeeper server")

	// Запускаем сервер
	if err := runServer(ctx, cfg, zapLogger); err != nil {
		zapLogger.Fatal("Server failed", zap.Error(err))
	}

	zapLogger.Info("Server shutdown complete")
}

// runServer запускает сервер с graceful shutdown.
func runServer(ctx context.Context, cfg *config.ServerConfig, logger *zap.Logger) error {
	// Инициализация базы данных
	dbStorage, err := storage.NewPostgresStorage(ctx, cfg.DatabaseURI, logger)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbStorage.Close()

	logger.Info("Connected to database")

	// Загружаем ключи шифрования
	privateKeyData, err := os.ReadFile("keys/private.pem")
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	publicKeyData, err := os.ReadFile("keys/public.pem")
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	// Инициализация сервисов
	authService := auth.NewService(cfg.JWTSecret)
	cryptoService, err := crypto.NewService(privateKeyData, publicKeyData)
	if err != nil {
		return fmt.Errorf("failed to create crypto service: %w", err)
	}
	otpService := otp.NewService()

	// Создание gRPC сервера
	gkServer := grpcServer.NewServer(dbStorage, authService, cryptoService, otpService, logger)

	// Создание компонентов сервера
	components, err := setupServerComponents(cfg, gkServer, authService, logger)
	if err != nil {
		return fmt.Errorf("failed to setup server components: %w", err)
	}

	// Запуск серверов
	startServers(components)

	logger.Info("All servers started successfully")

	// Ожидание сигнала завершения
	<-ctx.Done()
	logger.Info("Shutdown signal received")

	// Graceful shutdown с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := performGracefulShutdown(shutdownCtx, components); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
		return err
	}

	logger.Info("Server shutdown completed")
	return nil
}

// setupServerComponents создает все компоненты сервера.
func setupServerComponents(cfg *config.ServerConfig, gkServer *grpcServer.Server, authService *auth.Service, logger *zap.Logger) (*ServerComponents, error) {
	// Создаем HTTP сервер
	httpServer := &http.Server{
		Addr:         cfg.ServerAddress,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Настраиваем HTTP роуты
	router := setupHTTPRoutes(gkServer, authService, logger)
	httpServer.Handler = router

	// Создаем gRPC listener
	grpcListener, err := net.Listen("tcp", cfg.GRPCAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC listener: %w", err)
	}

	// Создаем gRPC сервер
	grpcSrv := grpcServer.NewGRPCServer(gkServer)
	reflection.Register(grpcSrv)

	return &ServerComponents{
		Storage:      nil, // Будет установлено позже
		HTTPServer:   httpServer,
		GRPCServer:   grpcSrv,
		GRPCListener: grpcListener,
		Logger:       logger,
	}, nil
}

// startServers запускает HTTP и gRPC серверы в горутинах.
func startServers(components *ServerComponents) {
	// Запускаем HTTP сервер в горутине
	go func() {
		components.Logger.Info("Starting HTTP server", zap.String("address", components.HTTPServer.Addr))
		if err := components.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			components.Logger.Error("HTTP server failed", zap.Error(err))
		}
	}()

	// Запускаем gRPC сервер в горутине
	go func() {
		components.Logger.Info("Starting gRPC server", zap.String("address", components.GRPCListener.Addr().String()))
		if err := components.GRPCServer.Serve(components.GRPCListener); err != nil {
			components.Logger.Error("gRPC server failed", zap.Error(err))
		}
	}()
}

// setupHTTPRoutes настраивает HTTP роуты для REST API.
func setupHTTPRoutes(gkServer *grpcServer.Server, authService *auth.Service, logger *zap.Logger) http.Handler {
	// Создаем роутер
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.CORSMiddleware())

	// Публичные роуты
	router.Post("/auth/register", gkServer.HandleRegister)
	router.Post("/auth/login", gkServer.HandleLogin)
	router.Post("/auth/refresh", gkServer.HandleRefreshToken)
	router.Post("/otp/generate", gkServer.HandleGenerateOTP)
	router.Post("/otp/secret", gkServer.HandleCreateOTPSecret)

	// Защищенные роуты
	router.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authService, logger))
		r.Get("/data", gkServer.HandleListData)
		r.Post("/data", gkServer.HandleCreateData)
		r.Get("/data/{id}", gkServer.HandleGetData)
		r.Put("/data/{id}", gkServer.HandleUpdateData)
		r.Delete("/data/{id}", gkServer.HandleDeleteData)
		r.Post("/sync", gkServer.HandleSyncData)
	})

	return router
}

// performGracefulShutdown корректно останавливает серверы.
func performGracefulShutdown(ctx context.Context, components *ServerComponents) error {
	components.Logger.Info("Starting graceful shutdown")

	// Останавливаем gRPC сервер первым (graceful stop)
	if components.GRPCServer != nil {
		components.Logger.Info("Stopping gRPC server...")
		go func() {
			components.GRPCServer.GracefulStop()
		}()

		// Устанавливаем таймаут для graceful stop
		go func() {
			<-ctx.Done()
			components.Logger.Info("Forcing gRPC server stop...")
			components.GRPCServer.Stop()
		}()
		components.Logger.Info("gRPC server stopped")
	}

	// Останавливаем HTTP сервер
	components.Logger.Info("Stopping HTTP server...")
	if err := components.HTTPServer.Shutdown(ctx); err != nil {
		components.Logger.Error("Error stopping HTTP server", zap.Error(err))
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}
	components.Logger.Info("HTTP server stopped")

	// Закрываем gRPC listener
	if components.GRPCListener != nil {
		if err := components.GRPCListener.Close(); err != nil {
			components.Logger.Error("Error closing gRPC listener", zap.Error(err))
		}
	}

	components.Logger.Info("Graceful shutdown completed")
	return nil
}
