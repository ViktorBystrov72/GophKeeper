//go:build integration

package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/GophKeeper/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	gozap "go.uber.org/zap"
)

func setupIntegrationTestStorage(t *testing.T) *PostgresStorage {
	dsn := os.Getenv("DATABASE_URI")
	if dsn == "" {
		t.Skip("DATABASE_URI не установлен - пропускаем интеграционный тест")
	}

	logger, _ := gozap.NewDevelopment()

	// Создаем подключение к базе данных с выполнением миграций
	storage, err := NewPostgresStorage(context.Background(), dsn, logger)
	require.NoError(t, err)
	return storage
}

func TestIntegrationCreateAndGetUser(t *testing.T) {
	s := setupIntegrationTestStorage(t)
	defer s.Close()

	user := &models.User{
		Username:     "integration_test_user_" + uuid.NewString(),
		PasswordHash: "hash",
	}
	err := s.CreateUser(context.Background(), user)
	require.NoError(t, err)
	require.NotEmpty(t, user.ID)

	fetched, err := s.GetUserByUsername(context.Background(), user.Username)
	require.NoError(t, err)
	require.Equal(t, user.Username, fetched.Username)

	fetchedByID, err := s.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user.Username, fetchedByID.Username)
}

func TestIntegrationCRUDDataEntry(t *testing.T) {
	s := setupIntegrationTestStorage(t)
	defer s.Close()

	user := &models.User{
		Username:     "integration_test_user2_" + uuid.NewString(),
		PasswordHash: "hash",
	}
	require.NoError(t, s.CreateUser(context.Background(), user))

	entry := &models.DataEntry{
		UserID:        user.ID,
		Type:          models.DataTypeText,
		Name:          "Integration Test Note",
		Description:   "desc",
		EncryptedData: []byte("secret"),
		Metadata:      "meta",
	}
	err := s.CreateDataEntry(context.Background(), entry)
	require.NoError(t, err)
	require.NotEmpty(t, entry.ID)

	fetched, err := s.GetDataEntry(context.Background(), user.ID, entry.ID)
	require.NoError(t, err)
	require.Equal(t, entry.Name, fetched.Name)

	entries, err := s.GetDataEntries(context.Background(), user.ID, nil)
	require.NoError(t, err)
	found := false
	for _, e := range entries {
		if e.ID == entry.ID {
			found = true
		}
	}
	require.True(t, found)

	// Обновляем
	entry.Description = "updated"
	err = s.UpdateDataEntry(context.Background(), entry)
	require.NoError(t, err)

	fetched, err = s.GetDataEntry(context.Background(), user.ID, entry.ID)
	require.NoError(t, err)
	require.Equal(t, "updated", fetched.Description)

	// Удаляем
	err = s.DeleteDataEntry(context.Background(), user.ID, entry.ID)
	require.NoError(t, err)

	_, err = s.GetDataEntry(context.Background(), user.ID, entry.ID)
	require.Error(t, err)
}

func TestIntegrationDataTypes(t *testing.T) {
	s := setupIntegrationTestStorage(t)
	defer s.Close()

	user := &models.User{
		Username:     "integration_test_user3_" + uuid.NewString(),
		PasswordHash: "hash",
	}
	require.NoError(t, s.CreateUser(context.Background(), user))

	// Тестируем учетные данные
	credsEntry := &models.DataEntry{
		UserID:        user.ID,
		Type:          models.DataTypeCredentials,
		Name:          "Test Credentials",
		Description:   "Login credentials",
		EncryptedData: []byte("encrypted_creds"),
		Metadata:      "website: example.com",
	}
	require.NoError(t, s.CreateDataEntry(context.Background(), credsEntry))

	// Тестируем бинарные данные
	binaryEntry := &models.DataEntry{
		UserID:        user.ID,
		Type:          models.DataTypeBinary,
		Name:          "Test Binary",
		Description:   "Binary file",
		EncryptedData: []byte("encrypted_binary"),
		Metadata:      "filename: document.pdf",
	}
	require.NoError(t, s.CreateDataEntry(context.Background(), binaryEntry))

	// Тестируем данные карты
	cardEntry := &models.DataEntry{
		UserID:        user.ID,
		Type:          models.DataTypeCard,
		Name:          "Test Card",
		Description:   "Credit card",
		EncryptedData: []byte("encrypted_card"),
		Metadata:      "bank: TestBank",
	}
	require.NoError(t, s.CreateDataEntry(context.Background(), cardEntry))

	// Получаем все записи
	entries, err := s.GetDataEntries(context.Background(), user.ID, nil)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	// Получаем записи по типу
	textType := models.DataTypeText
	textEntries, err := s.GetDataEntries(context.Background(), user.ID, &textType)
	require.NoError(t, err)
	require.Len(t, textEntries, 0) // Мы не создавали текстовые записи

	credsType := models.DataTypeCredentials
	credsEntries, err := s.GetDataEntries(context.Background(), user.ID, &credsType)
	require.NoError(t, err)
	require.Len(t, credsEntries, 1)
	require.Equal(t, "Test Credentials", credsEntries[0].Name)
}

func TestIntegrationSyncData(t *testing.T) {
	s := setupIntegrationTestStorage(t)
	defer s.Close()

	user := &models.User{
		Username:     "integration_test_user4_" + uuid.NewString(),
		PasswordHash: "hash",
	}
	require.NoError(t, s.CreateUser(context.Background(), user))

	// Создаем начальные данные
	entry1 := &models.DataEntry{
		UserID:        user.ID,
		Type:          models.DataTypeText,
		Name:          "Sync Test 1",
		Description:   "First entry",
		EncryptedData: []byte("data1"),
		Metadata:      "meta1",
	}
	require.NoError(t, s.CreateDataEntry(context.Background(), entry1))

	// Ждем немного для обеспечения разных временных меток
	time.Sleep(100 * time.Millisecond)

	entry2 := &models.DataEntry{
		UserID:        user.ID,
		Type:          models.DataTypeText,
		Name:          "Sync Test 2",
		Description:   "Second entry",
		EncryptedData: []byte("data2"),
		Metadata:      "meta2",
	}
	require.NoError(t, s.CreateDataEntry(context.Background(), entry2))

	// Тестируем синхронизацию - получаем записи после определенного времени
	afterTime := entry1.CreatedAt
	entries, err := s.GetDataEntriesAfter(context.Background(), user.ID, afterTime)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "Sync Test 2", entries[0].Name)

	// Тестируем синхронизацию удаленных записей
	err = s.DeleteDataEntry(context.Background(), user.ID, entry1.ID)
	require.NoError(t, err)

	deletedEntries, err := s.GetDeletedEntriesAfter(context.Background(), user.ID, afterTime)
	require.NoError(t, err)
	require.Len(t, deletedEntries, 1)
	require.Equal(t, entry1.ID, deletedEntries[0])
}
