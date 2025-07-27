package storage

import (
	"context"
	"os"
	"testing"

	"github.com/GophKeeper/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	gozap "go.uber.org/zap"
)

func setupTestStorage(t *testing.T) *PostgresStorage {
	dsn := os.Getenv("DATABASE_URI")
	if dsn == "" {
		t.Skip("DATABASE_URI not set")
	}
	logger, _ := gozap.NewDevelopment()

	// Создаем подключение к базе данных с выполнением миграций
	storage, err := NewPostgresStorage(context.Background(), dsn, logger)
	require.NoError(t, err)
	return storage
}

func TestCreateAndGetUser(t *testing.T) {
	s := setupTestStorage(t)
	defer s.Close()

	user := &models.User{
		Username:     "testuser_" + uuid.NewString(),
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

func TestCRUDDataEntry(t *testing.T) {
	s := setupTestStorage(t)
	defer s.Close()

	user := &models.User{
		Username:     "testuser2_" + uuid.NewString(),
		PasswordHash: "hash",
	}
	require.NoError(t, s.CreateUser(context.Background(), user))

	entry := &models.DataEntry{
		UserID:        user.ID,
		Type:          models.DataTypeText,
		Name:          "Test Note",
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

	// Update
	entry.Description = "updated"
	err = s.UpdateDataEntry(context.Background(), entry)
	require.NoError(t, err)

	fetched, err = s.GetDataEntry(context.Background(), user.ID, entry.ID)
	require.NoError(t, err)
	require.Equal(t, "updated", fetched.Description)

	// Delete
	err = s.DeleteDataEntry(context.Background(), user.ID, entry.ID)
	require.NoError(t, err)

	_, err = s.GetDataEntry(context.Background(), user.ID, entry.ID)
	require.Error(t, err)
}
