package grpc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/GophKeeper/internal/auth"
	"github.com/GophKeeper/internal/crypto"
	"github.com/GophKeeper/internal/models"
	"github.com/GophKeeper/internal/otp"
	"github.com/GophKeeper/internal/storage"
	pb "github.com/GophKeeper/proto/gen/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRegister(t *testing.T) {
	client := setupTestClient(t)

	resp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: "testuser",
		Password: "testpass123",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Token)
	require.NotNil(t, resp.User)
	require.Equal(t, "testuser", resp.User.Username)
}

func TestLogin(t *testing.T) {
	client := setupTestClient(t)

	// Сначала регистрируем пользователя
	_, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: "testuser",
		Password: "testpass123",
	})
	require.NoError(t, err)

	// Затем входим
	resp, err := client.Login(context.Background(), &pb.LoginRequest{
		Username: "testuser",
		Password: "testpass123",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Token)
	require.NotNil(t, resp.User)
	require.Equal(t, "testuser", resp.User.Username)
}

func TestCreateData(t *testing.T) {
	client := setupTestClient(t)

	// Сначала регистрируем и входим
	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: "testuser",
		Password: "testpass123",
	})
	require.NoError(t, err)
	t.Logf("Register token: %s", loginResp.Token)

	// Создаем контекст с токеном
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Создаем данные
	resp, err := client.CreateData(ctx, &pb.CreateDataRequest{
		Type:          pb.DataType_DATA_TYPE_CREDENTIALS,
		Name:          "Test Credentials",
		Description:   "Test description",
		EncryptedData: []byte("encrypted-data"),
		Metadata:      "test-metadata",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.DataEntry)
	require.Equal(t, "Test Credentials", resp.DataEntry.Name)
}

func TestListData(t *testing.T) {
	client := setupTestClient(t)

	// Сначала регистрируем и входим
	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: "testuser",
		Password: "testpass123",
	})
	require.NoError(t, err)

	// Создаем контекст с токеном
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Получаем список данных
	resp, err := client.ListData(ctx, &pb.ListDataRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.DataEntries, 0) // Изначально список пустой
}

func TestRefreshToken(t *testing.T) {
	client := setupTestClient(t)

	// Сначала регистрируем и входим
	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: "testuser",
		Password: "testpass123",
	})
	require.NoError(t, err)

	// Обновляем токен
	resp, err := client.RefreshToken(context.Background(), &pb.RefreshTokenRequest{
		Token: loginResp.Token,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Token)
}

func TestGenerateOTP(t *testing.T) {
	client := setupTestClient(t)

	secret := "JBSWY3DPEHPK3PXP"
	resp, err := client.GenerateOTP(context.Background(), &pb.GenerateOTPRequest{
		Secret: secret,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Code)
	require.Len(t, resp.Code, 6)
	require.Greater(t, resp.TimeRemaining, int32(0))
}

func TestCreateOTPSecret(t *testing.T) {
	client := setupTestClient(t)

	resp, err := client.CreateOTPSecret(context.Background(), &pb.CreateOTPSecretRequest{
		Issuer:      "GophKeeper",
		AccountName: "test@example.com",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Secret)
	require.NotEmpty(t, resp.QrCodeUrl)
	require.Len(t, resp.BackupCodes, 10)
}

func TestUnauthorizedAccess(t *testing.T) {
	client := setupTestClient(t)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer invalid-token")

	_, err := client.ListData(ctx, &pb.ListDataRequest{})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unauthenticated, st.Code())
}

func TestInvalidCredentials(t *testing.T) {
	client := setupTestClient(t)

	_, err := client.Login(context.Background(), &pb.LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unauthenticated, st.Code())
}

// setupTestClient создает тестовый клиент с собственным mockStorage
func setupTestClient(t *testing.T) pb.GophKeeperClient {
	// Настройка тестового окружения
	logger, _ := zap.NewDevelopment()

	// Генерируем тестовые ключи программно
	privateKey, publicKey := generateTestKeys(t)

	// Создаем сервисы
	authService := auth.NewService("test-secret")
	cryptoService, err := crypto.NewService(privateKey, publicKey)
	require.NoError(t, err)
	otpService := otp.NewService()

	// Создаем тестовое хранилище (in-memory для тестов)
	storage := setupTestStorage(t)

	// Создаем сервер
	server := NewServer(storage, authService, cryptoService, otpService, logger)

	// Запускаем тестовый сервер
	lis, err := startTestGRPCServer(server, "localhost:0", logger)
	require.NoError(t, err)
	t.Cleanup(func() {
		lis.Close()
	})

	// Подключаемся к серверу
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Close()
	})

	return pb.NewGophKeeperClient(conn)
}

// setupTestStorage создает тестовое хранилище
func setupTestStorage(t *testing.T) storage.Storage {
	// Для интеграционных тестов используем in-memory хранилище
	// В реальном проекте здесь можно использовать тестовую базу данных
	return &mockStorage{}
}

// mockStorage - простое in-memory хранилище для тестов
type mockStorage struct {
	users map[string]*models.User
	data  map[uuid.UUID]*models.DataEntry
}

func (m *mockStorage) CreateUser(ctx context.Context, user *models.User) error {
	if m.users == nil {
		m.users = make(map[string]*models.User)
	}
	// Убеждаемся, что у пользователя есть ID
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	m.users[user.Username] = user
	return nil
}

func (m *mockStorage) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	if user, exists := m.users[username]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *mockStorage) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	for _, user := range m.users {
		if user.ID == userID {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *mockStorage) CreateDataEntry(ctx context.Context, entry *models.DataEntry) error {
	if m.data == nil {
		m.data = make(map[uuid.UUID]*models.DataEntry)
	}
	m.data[entry.ID] = entry
	return nil
}

func (m *mockStorage) GetDataEntry(ctx context.Context, userID, entryID uuid.UUID) (*models.DataEntry, error) {
	if entry, exists := m.data[entryID]; exists && entry.UserID == userID {
		return entry, nil
	}
	return nil, fmt.Errorf("data entry not found")
}

func (m *mockStorage) GetDataEntries(ctx context.Context, userID uuid.UUID, dataType *models.DataType) ([]models.DataEntry, error) {
	var entries []models.DataEntry
	for _, entry := range m.data {
		if entry.UserID == userID {
			if dataType == nil || entry.Type == *dataType {
				entries = append(entries, *entry)
			}
		}
	}
	return entries, nil
}

func (m *mockStorage) UpdateDataEntry(ctx context.Context, entry *models.DataEntry) error {
	if _, exists := m.data[entry.ID]; exists {
		m.data[entry.ID] = entry
		return nil
	}
	return fmt.Errorf("data entry not found")
}

func (m *mockStorage) DeleteDataEntry(ctx context.Context, userID, entryID uuid.UUID) error {
	if entry, exists := m.data[entryID]; exists && entry.UserID == userID {
		delete(m.data, entryID)
		return nil
	}
	return fmt.Errorf("data entry not found")
}

func (m *mockStorage) GetDataEntriesAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]models.DataEntry, error) {
	var entries []models.DataEntry
	for _, entry := range m.data {
		if entry.UserID == userID && entry.UpdatedAt.After(after) {
			entries = append(entries, *entry)
		}
	}
	return entries, nil
}

func (m *mockStorage) GetDeletedEntriesAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]uuid.UUID, error) {
	// Простая реализация для тестов
	return []uuid.UUID{}, nil
}

func (m *mockStorage) Close() {
	// Ничего не делаем для in-memory хранилища
}

// startTestGRPCServer запускает тестовый gRPC сервер
func startTestGRPCServer(srv *Server, addr string, logger *zap.Logger) (lisCloser, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(AuthInterceptor(srv.authService, srv.logger)),
	)
	pb.RegisterGophKeeperServer(grpcServer, srv)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("Failed to serve gRPC", zap.Error(err))
		}
	}()

	return &testListener{lis: lis, server: grpcServer}, nil
}

// generateTestKeys создает тестовые RSA ключи для тестов
func generateTestKeys(t *testing.T) ([]byte, []byte) {
	// Генерируем приватный ключ
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Кодируем приватный ключ в PEM
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	// Извлекаем публичный ключ
	publicKey := &privateKey.PublicKey
	publicKeyDER, err := x509.MarshalPKIXPublicKey(publicKey)
	require.NoError(t, err)
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	}
	publicKeyBytes := pem.EncodeToMemory(publicKeyPEM)

	return privateKeyBytes, publicKeyBytes
}

type testListener struct {
	lis    net.Listener
	server *grpc.Server
}

func (t *testListener) Close() error {
	t.server.Stop()
	return t.lis.Close()
}

func (t *testListener) Addr() net.Addr {
	return t.lis.Addr()
}

type lisCloser interface {
	Close() error
	Addr() net.Addr
}
