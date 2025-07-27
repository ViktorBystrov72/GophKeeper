// Package client предоставляет клиентскую часть для GophKeeper.
package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/GophKeeper/internal/config"
	pb "github.com/GophKeeper/proto/gen/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client представляет клиент для взаимодействия с сервером GophKeeper.
type Client struct {
	config     *config.ClientConfig
	logger     *zap.Logger
	conn       *grpc.ClientConn
	grpcClient pb.GophKeeperClient
	token      string
	expiresAt  time.Time
}

// NewClient создает новый клиент GophKeeper.
func NewClient(cfg *config.ClientConfig, logger *zap.Logger) (*Client, error) {
	var opts []grpc.DialOption

	if cfg.EnableTLS {
		var creds credentials.TransportCredentials
		if cfg.CertFile != "" {
			// Загружаем сертификат из файла
			tlsConfig := &tls.Config{
				ServerName: "localhost",
			}
			creds = credentials.NewTLS(tlsConfig)
		} else {
			// Используем системные сертификаты
			creds = credentials.NewTLS(&tls.Config{})
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Подключение к серверу
	conn, err := grpc.NewClient(cfg.GRPCAddress, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	grpcClient := pb.NewGophKeeperClient(conn)

	return &Client{
		config:     cfg,
		logger:     logger,
		conn:       conn,
		grpcClient: grpcClient,
	}, nil
}

// Close закрывает соединение с сервером.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Register регистрирует нового пользователя.
func (c *Client) Register(ctx context.Context, username, password string) error {
	req := &pb.RegisterRequest{
		Username: username,
		Password: password,
	}

	resp, err := c.grpcClient.Register(ctx, req)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Сохраняем токен
	c.token = resp.Token
	c.expiresAt = resp.ExpiresAt.AsTime()

	c.logger.Info("Successfully registered and logged in",
		zap.String("username", username))

	return nil
}

// Login выполняет аутентификацию пользователя.
func (c *Client) Login(ctx context.Context, username, password string) error {
	req := &pb.LoginRequest{
		Username: username,
		Password: password,
	}

	resp, err := c.grpcClient.Login(ctx, req)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Сохраняем токен
	c.token = resp.Token
	c.expiresAt = resp.ExpiresAt.AsTime()

	c.logger.Info("Successfully logged in",
		zap.String("username", username))

	return nil
}

// IsAuthenticated проверяет, аутентифицирован ли пользователь.
func (c *Client) IsAuthenticated() bool {
	return c.token != "" && time.Now().Before(c.expiresAt)
}

// RefreshToken обновляет токен аутентификации.
func (c *Client) RefreshToken(ctx context.Context) error {
	if c.token == "" {
		return fmt.Errorf("no token to refresh")
	}

	req := &pb.RefreshTokenRequest{
		Token: c.token,
	}

	resp, err := c.grpcClient.RefreshToken(ctx, req)
	if err != nil {
		return fmt.Errorf("token refresh failed: %w", err)
	}

	c.token = resp.Token
	c.expiresAt = resp.ExpiresAt.AsTime()

	c.logger.Debug("Token refreshed successfully")
	return nil
}

// CreateData создает новую запись данных.
func (c *Client) CreateData(ctx context.Context, req *pb.CreateDataRequest) (*pb.DataEntry, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	ctx = c.addAuthToContext(ctx)
	resp, err := c.grpcClient.CreateData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create data: %w", err)
	}

	return resp.DataEntry, nil
}

// GetData получает запись данных по ID.
func (c *Client) GetData(ctx context.Context, id string) (*pb.DataEntry, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	req := &pb.GetDataRequest{Id: id}
	ctx = c.addAuthToContext(ctx)

	resp, err := c.grpcClient.GetData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get data: %w", err)
	}

	return resp.DataEntry, nil
}

// ListData получает список записей данных.
func (c *Client) ListData(ctx context.Context, dataType *pb.DataType) ([]*pb.DataEntry, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	req := &pb.ListDataRequest{
		Type: dataType,
	}
	ctx = c.addAuthToContext(ctx)

	resp, err := c.grpcClient.ListData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list data: %w", err)
	}

	return resp.DataEntries, nil
}

// UpdateData обновляет запись данных.
func (c *Client) UpdateData(ctx context.Context, req *pb.UpdateDataRequest) (*pb.DataEntry, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	ctx = c.addAuthToContext(ctx)
	resp, err := c.grpcClient.UpdateData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update data: %w", err)
	}

	return resp.DataEntry, nil
}

// DeleteData удаляет запись данных.
func (c *Client) DeleteData(ctx context.Context, id string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("not authenticated")
	}

	req := &pb.DeleteDataRequest{Id: id}
	ctx = c.addAuthToContext(ctx)

	_, err := c.grpcClient.DeleteData(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete data: %w", err)
	}

	return nil
}

// SyncData синхронизирует данные с сервером.
func (c *Client) SyncData(ctx context.Context, lastSyncTime time.Time) (*pb.SyncDataResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	req := &pb.SyncDataRequest{
		LastSyncTime: timestamppb.New(lastSyncTime),
	}
	ctx = c.addAuthToContext(ctx)

	resp, err := c.grpcClient.SyncData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to sync data: %w", err)
	}

	return resp, nil
}

// GenerateOTP генерирует OTP код.
func (c *Client) GenerateOTP(ctx context.Context, secret string) (*pb.GenerateOTPResponse, error) {
	req := &pb.GenerateOTPRequest{Secret: secret}

	resp, err := c.grpcClient.GenerateOTP(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP: %w", err)
	}

	return resp, nil
}

// CreateOTPSecret создает новый OTP секрет.
func (c *Client) CreateOTPSecret(ctx context.Context, issuer, accountName string) (*pb.CreateOTPSecretResponse, error) {
	req := &pb.CreateOTPSecretRequest{
		Issuer:      issuer,
		AccountName: accountName,
	}

	resp, err := c.grpcClient.CreateOTPSecret(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTP secret: %w", err)
	}

	return resp, nil
}

// addAuthToContext добавляет токен аутентификации в контекст.
func (c *Client) addAuthToContext(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.token)
}
