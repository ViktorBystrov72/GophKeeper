//go:build integration

package grpc

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/GophKeeper/internal/auth"
	"github.com/GophKeeper/internal/crypto"
	"github.com/GophKeeper/internal/migrations"
	"github.com/GophKeeper/internal/otp"
	"github.com/GophKeeper/internal/storage"
	pb "github.com/GophKeeper/proto/gen/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func setupIntegrationTestClient(t *testing.T) pb.GophKeeperClient {
	dsn := os.Getenv("DATABASE_URI")
	if dsn == "" {
		t.Skip("DATABASE_URI не установлен - пропускаем интеграционный тест")
	}

	// Настройка тестового окружения
	logger, _ := zap.NewDevelopment()

	// Выполняем миграции перед созданием хранилища
	err := migrations.RunMigrations(context.Background(), dsn, "../../migrations")
	require.NoError(t, err)

	// Генерируем тестовые ключи программно
	privateKey, publicKey := generateTestKeys(t)

	// Создаем сервисы
	authService := auth.NewService("test-secret")
	cryptoService, err := crypto.NewService(privateKey, publicKey)
	require.NoError(t, err)
	otpService := otp.NewService()

	// Создаем реальное хранилище с базой данных
	storage, err := storage.NewPostgresStorage(context.Background(), dsn, logger)
	require.NoError(t, err)

	// Создаем сервер
	server := NewServer(storage, authService, cryptoService, otpService, logger)

	// Запускаем тестовый сервер
	lis, err := startTestGRPCServer(server, "localhost:0", logger)
	require.NoError(t, err)
	t.Cleanup(func() {
		lis.Close()
		storage.Close()
	})

	// Подключаемся к серверу
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Close()
	})

	return pb.NewGophKeeperClient(conn)
}

func TestIntegrationRegisterAndLogin(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "integration_test_user_" + uuid.NewString()
	password := "testpass123"

	// Регистрация
	resp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Token)
	require.NotNil(t, resp.User)
	require.Equal(t, username, resp.User.Username)

	// Вход
	loginResp, err := client.Login(context.Background(), &pb.LoginRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)
	require.NotEmpty(t, loginResp.Token)
	require.NotNil(t, loginResp.User)
	require.Equal(t, username, loginResp.User.Username)
}

func TestIntegrationCreateAndGetData(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "integration_test_user2_" + uuid.NewString()
	password := "testpass123"

	// Регистрируемся и входим
	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	// Создаем контекст с токеном
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Создаем данные
	createResp, err := client.CreateData(ctx, &pb.CreateDataRequest{
		Type:          pb.DataType_DATA_TYPE_CREDENTIALS,
		Name:          "Integration Test Credentials",
		Description:   "Test description",
		EncryptedData: []byte("encrypted-data"),
		Metadata:      "test-metadata",
	})
	require.NoError(t, err)
	require.NotNil(t, createResp.DataEntry)
	require.Equal(t, "Integration Test Credentials", createResp.DataEntry.Name)

	// Получаем список данных
	listResp, err := client.ListData(ctx, &pb.ListDataRequest{})
	require.NoError(t, err)
	require.NotNil(t, listResp)
	require.Len(t, listResp.DataEntries, 1)
	require.Equal(t, "Integration Test Credentials", listResp.DataEntries[0].Name)

	// Получаем конкретную запись
	getResp, err := client.GetData(ctx, &pb.GetDataRequest{
		Id: createResp.DataEntry.Id,
	})
	require.NoError(t, err)
	require.NotNil(t, getResp.DataEntry)
	require.Equal(t, "Integration Test Credentials", getResp.DataEntry.Name)
}

func TestIntegrationUpdateAndDeleteData(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "integration_test_user3_" + uuid.NewString()
	password := "testpass123"

	// Регистрируемся и входим
	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Создаем данные
	createResp, err := client.CreateData(ctx, &pb.CreateDataRequest{
		Type:          pb.DataType_DATA_TYPE_TEXT,
		Name:          "Test Text Data",
		Description:   "Original description",
		EncryptedData: []byte("original-data"),
		Metadata:      "original-metadata",
	})
	require.NoError(t, err)

	// Обновляем данные
	updateResp, err := client.UpdateData(ctx, &pb.UpdateDataRequest{
		Id:            createResp.DataEntry.Id,
		Name:          "Updated Text Data",
		Description:   "Updated description",
		EncryptedData: []byte("updated-data"),
		Metadata:      "updated-metadata",
		Version:       createResp.DataEntry.Version,
	})
	require.NoError(t, err)
	require.NotNil(t, updateResp.DataEntry)
	require.Equal(t, "Updated Text Data", updateResp.DataEntry.Name)
	require.Equal(t, "Updated description", updateResp.DataEntry.Description)

	// Удаляем данные
	deleteResp, err := client.DeleteData(ctx, &pb.DeleteDataRequest{
		Id: createResp.DataEntry.Id,
	})
	require.NoError(t, err)
	require.True(t, deleteResp.Success)

	// Проверяем, что данные удалены
	listResp, err := client.ListData(ctx, &pb.ListDataRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.DataEntries, 0)
}

func TestIntegrationSyncData(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "integration_test_user4_" + uuid.NewString()
	password := "testpass123"

	// Регистрируемся и входим
	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Создаем несколько записей
	for i := 1; i <= 3; i++ {
		_, err := client.CreateData(ctx, &pb.CreateDataRequest{
			Type:          pb.DataType_DATA_TYPE_TEXT,
			Name:          fmt.Sprintf("Sync Test %d", i),
			Description:   fmt.Sprintf("Description %d", i),
			EncryptedData: []byte(fmt.Sprintf("data-%d", i)),
			Metadata:      fmt.Sprintf("meta-%d", i),
		})
		require.NoError(t, err)
	}

	// Синхронизируем данные
	lastSyncTime := timestamppb.New(time.Now().Add(-time.Hour))
	syncResp, err := client.SyncData(ctx, &pb.SyncDataRequest{
		LastSyncTime: lastSyncTime,
	})
	require.NoError(t, err)
	require.NotNil(t, syncResp)
	require.Len(t, syncResp.DataEntries, 3)
}

func TestIntegrationOTP(t *testing.T) {
	client := setupIntegrationTestClient(t)

	// Создаем OTP секрет
	secretResp, err := client.CreateOTPSecret(context.Background(), &pb.CreateOTPSecretRequest{
		Issuer:      "Integration Test",
		AccountName: "test@example.com",
	})
	require.NoError(t, err)
	require.NotEmpty(t, secretResp.Secret)
	require.NotEmpty(t, secretResp.QrCodeUrl)
	require.Len(t, secretResp.BackupCodes, 10)

	// Генерируем OTP код
	codeResp, err := client.GenerateOTP(context.Background(), &pb.GenerateOTPRequest{
		Secret: secretResp.Secret,
	})
	require.NoError(t, err)
	require.NotEmpty(t, codeResp.Code)
	require.Len(t, codeResp.Code, 6)
	require.Greater(t, codeResp.TimeRemaining, int32(0))
}
