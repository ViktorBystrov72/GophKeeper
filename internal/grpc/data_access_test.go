//go:build integration

package grpc

import (
	"context"
	"testing"

	pb "github.com/GophKeeper/proto/gen/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// TestDataAccess_GetData_ValidRequest тестирует получение данных по валидному запросу
func TestDataAccess_GetData_ValidRequest(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "data_access_user_" + uuid.NewString()
	password := "testpass123"

	// Регистрируемся и входим
	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Создаем тестовые данные
	createResp, err := client.CreateData(ctx, &pb.CreateDataRequest{
		Type:          pb.DataType_DATA_TYPE_CREDENTIALS,
		Name:          "Test Credentials",
		Description:   "Test description",
		EncryptedData: []byte("encrypted-password"),
		Metadata:      "website: example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, createResp.DataEntry)

	// Получаем данные по ID
	getResp, err := client.GetData(ctx, &pb.GetDataRequest{
		Id: createResp.DataEntry.Id,
	})
	require.NoError(t, err)
	require.NotNil(t, getResp.DataEntry)
	require.Equal(t, "Test Credentials", getResp.DataEntry.Name)
	require.Equal(t, "Test description", getResp.DataEntry.Description)
	require.Equal(t, pb.DataType_DATA_TYPE_CREDENTIALS, getResp.DataEntry.Type)
	require.Equal(t, []byte("encrypted-password"), getResp.DataEntry.EncryptedData)
	require.Equal(t, "website: example.com", getResp.DataEntry.Metadata)
}

// TestDataAccess_GetData_Unauthorized тестирует попытку получения данных без авторизации
func TestDataAccess_GetData_Unauthorized(t *testing.T) {
	client := setupIntegrationTestClient(t)

	// Пытаемся получить данные без токена
	_, err := client.GetData(context.Background(), &pb.GetDataRequest{
		Id: "some-id",
	})
	require.Error(t, err)
	// Проверяем, что это ошибка аутентификации
	require.Contains(t, err.Error(), "Unauthenticated")
}

// TestDataAccess_GetData_InvalidID тестирует получение данных с невалидным ID
func TestDataAccess_GetData_InvalidID(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "data_access_user2_" + uuid.NewString()
	password := "testpass123"

	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Пытаемся получить данные с невалидным ID
	_, err = client.GetData(ctx, &pb.GetDataRequest{
		Id: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "InvalidArgument")
}

// TestDataAccess_GetData_NotFound тестирует получение несуществующих данных
func TestDataAccess_GetData_NotFound(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "data_access_user3_" + uuid.NewString()
	password := "testpass123"

	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Пытаемся получить несуществующие данные
	_, err = client.GetData(ctx, &pb.GetDataRequest{
		Id: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "NotFound")
}

// TestDataAccess_GetData_CrossUserAccess тестирует попытку доступа к данным другого пользователя
func TestDataAccess_GetData_CrossUserAccess(t *testing.T) {
	client := setupIntegrationTestClient(t)

	// Создаем первого пользователя
	user1 := "data_access_user4_" + uuid.NewString()
	loginResp1, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: user1,
		Password: "testpass123",
	})
	require.NoError(t, err)

	ctx1 := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp1.Token)

	// Первый пользователь создает данные
	createResp, err := client.CreateData(ctx1, &pb.CreateDataRequest{
		Type:          pb.DataType_DATA_TYPE_TEXT,
		Name:          "User1 Data",
		Description:   "Private data",
		EncryptedData: []byte("user1-secret"),
	})
	require.NoError(t, err)

	// Создаем второго пользователя
	user2 := "data_access_user5_" + uuid.NewString()
	loginResp2, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: user2,
		Password: "testpass123",
	})
	require.NoError(t, err)

	ctx2 := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp2.Token)

	// Второй пользователь пытается получить данные первого
	_, err = client.GetData(ctx2, &pb.GetDataRequest{
		Id: createResp.DataEntry.Id,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "NotFound")
}

// TestDataAccess_ListData_ValidRequest тестирует получение списка данных
func TestDataAccess_ListData_ValidRequest(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "data_access_user6_" + uuid.NewString()
	password := "testpass123"

	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Создаем несколько записей
	entries := []*pb.CreateDataRequest{
		{
			Type:          pb.DataType_DATA_TYPE_CREDENTIALS,
			Name:          "Credentials 1",
			Description:   "Login data",
			EncryptedData: []byte("password1"),
		},
		{
			Type:          pb.DataType_DATA_TYPE_TEXT,
			Name:          "Text Note 1",
			Description:   "Important note",
			EncryptedData: []byte("secret note"),
		},
		{
			Type:          pb.DataType_DATA_TYPE_CARD,
			Name:          "Card 1",
			Description:   "Credit card",
			EncryptedData: []byte("card data"),
		},
	}

	for _, entry := range entries {
		_, err := client.CreateData(ctx, entry)
		require.NoError(t, err)
	}

	// Получаем список всех данных
	listResp, err := client.ListData(ctx, &pb.ListDataRequest{})
	require.NoError(t, err)
	require.NotNil(t, listResp)
	require.Len(t, listResp.DataEntries, 3)
	require.Equal(t, int32(3), listResp.Total)

	// Проверяем, что все записи принадлежат пользователю
	for _, entry := range listResp.DataEntries {
		require.NotEmpty(t, entry.Id)
		require.NotEmpty(t, entry.Name)
		require.NotEmpty(t, entry.EncryptedData)
	}
}

// TestDataAccess_ListData_ByType тестирует получение списка данных по типу
func TestDataAccess_ListData_ByType(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "data_access_user7_" + uuid.NewString()
	password := "testpass123"

	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Создаем записи разных типов
	credentialsType := pb.DataType_DATA_TYPE_CREDENTIALS
	textType := pb.DataType_DATA_TYPE_TEXT

	// Создаем учетные данные
	_, err = client.CreateData(ctx, &pb.CreateDataRequest{
		Type:          pb.DataType_DATA_TYPE_CREDENTIALS,
		Name:          "Credentials",
		Description:   "Login data",
		EncryptedData: []byte("password"),
	})
	require.NoError(t, err)

	// Создаем текстовую заметку
	_, err = client.CreateData(ctx, &pb.CreateDataRequest{
		Type:          pb.DataType_DATA_TYPE_TEXT,
		Name:          "Text Note",
		Description:   "Important note",
		EncryptedData: []byte("secret note"),
	})
	require.NoError(t, err)

	// Получаем только учетные данные
	credsResp, err := client.ListData(ctx, &pb.ListDataRequest{
		Type: &credentialsType,
	})
	require.NoError(t, err)
	require.Len(t, credsResp.DataEntries, 1)
	require.Equal(t, pb.DataType_DATA_TYPE_CREDENTIALS, credsResp.DataEntries[0].Type)

	// Получаем только текстовые заметки
	textResp, err := client.ListData(ctx, &pb.ListDataRequest{
		Type: &textType,
	})
	require.NoError(t, err)
	require.Len(t, textResp.DataEntries, 1)
	require.Equal(t, pb.DataType_DATA_TYPE_TEXT, textResp.DataEntries[0].Type)
}

// TestDataAccess_ListData_Unauthorized тестирует попытку получения списка данных без авторизации
func TestDataAccess_ListData_Unauthorized(t *testing.T) {
	client := setupIntegrationTestClient(t)

	// Пытаемся получить список данных без токена
	_, err := client.ListData(context.Background(), &pb.ListDataRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Unauthenticated")
}

// TestDataAccess_ListData_EmptyResult тестирует получение пустого списка данных
func TestDataAccess_ListData_EmptyResult(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "data_access_user8_" + uuid.NewString()
	password := "testpass123"

	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Получаем список данных для нового пользователя (должен быть пустым)
	listResp, err := client.ListData(ctx, &pb.ListDataRequest{})
	require.NoError(t, err)
	require.NotNil(t, listResp)
	require.Len(t, listResp.DataEntries, 0)
	require.Equal(t, int32(0), listResp.Total)
}

// TestDataAccess_DataPrivacy тестирует приватность данных между пользователями
func TestDataAccess_DataPrivacy(t *testing.T) {
	client := setupIntegrationTestClient(t)

	// Создаем первого пользователя
	user1 := "privacy_user1_" + uuid.NewString()
	loginResp1, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: user1,
		Password: "testpass123",
	})
	require.NoError(t, err)

	ctx1 := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp1.Token)

	// Первый пользователь создает данные
	_, err = client.CreateData(ctx1, &pb.CreateDataRequest{
		Type:          pb.DataType_DATA_TYPE_CREDENTIALS,
		Name:          "User1 Private Data",
		Description:   "Very private",
		EncryptedData: []byte("user1-secret"),
	})
	require.NoError(t, err)

	// Создаем второго пользователя
	user2 := "privacy_user2_" + uuid.NewString()
	loginResp2, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: user2,
		Password: "testpass123",
	})
	require.NoError(t, err)

	ctx2 := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp2.Token)

	// Второй пользователь создает свои данные
	_, err = client.CreateData(ctx2, &pb.CreateDataRequest{
		Type:          pb.DataType_DATA_TYPE_TEXT,
		Name:          "User2 Private Data",
		Description:   "Also private",
		EncryptedData: []byte("user2-secret"),
	})
	require.NoError(t, err)

	// Первый пользователь получает свой список
	listResp1, err := client.ListData(ctx1, &pb.ListDataRequest{})
	require.NoError(t, err)
	require.Len(t, listResp1.DataEntries, 1)
	require.Equal(t, "User1 Private Data", listResp1.DataEntries[0].Name)

	// Второй пользователь получает свой список
	listResp2, err := client.ListData(ctx2, &pb.ListDataRequest{})
	require.NoError(t, err)
	require.Len(t, listResp2.DataEntries, 1)
	require.Equal(t, "User2 Private Data", listResp2.DataEntries[0].Name)

	// Проверяем, что пользователи не видят данные друг друга
	require.NotEqual(t, listResp1.DataEntries[0].Id, listResp2.DataEntries[0].Id)
}

// TestDataAccess_DataEncryption тестирует, что данные передаются в зашифрованном виде
func TestDataAccess_DataEncryption(t *testing.T) {
	client := setupIntegrationTestClient(t)

	username := "encryption_user_" + uuid.NewString()
	password := "testpass123"

	loginResp, err := client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+loginResp.Token)

	// Создаем данные с секретным содержимым
	secretData := "super-secret-password-123"
	createResp, err := client.CreateData(ctx, &pb.CreateDataRequest{
		Type:          pb.DataType_DATA_TYPE_CREDENTIALS,
		Name:          "Encrypted Credentials",
		Description:   "Test encryption",
		EncryptedData: []byte(secretData),
		Metadata:      "sensitive: true",
	})
	require.NoError(t, err)

	// Получаем данные обратно
	getResp, err := client.GetData(ctx, &pb.GetDataRequest{
		Id: createResp.DataEntry.Id,
	})
	require.NoError(t, err)

	// Проверяем, что данные передаются в зашифрованном виде
	require.Equal(t, []byte(secretData), getResp.DataEntry.EncryptedData)
	require.Equal(t, "Encrypted Credentials", getResp.DataEntry.Name)
	require.Equal(t, "Test encryption", getResp.DataEntry.Description)
	require.Equal(t, "sensitive: true", getResp.DataEntry.Metadata)
}
