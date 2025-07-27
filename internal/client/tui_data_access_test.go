package client

import (
	"testing"

	pb "github.com/GophKeeper/proto/gen/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestTUI_DataAccess_ViewDataList тестирует просмотр списка данных в TUI
func TestTUI_DataAccess_ViewDataList(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Создаем тестовые данные в формате pb.DataEntry
	entries := []*pb.DataEntry{
		{Id: "1", Name: "Test Credentials", Description: "Login data"},
		{Id: "2", Name: "Test Note", Description: "Important note"},
	}

	msg := dataListMsg{entries: entries}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, 2, len(updatedModel.list.Items()))
	assert.Nil(t, cmd)
}

// TestTUI_DataAccess_ViewSpecificData тестирует просмотр конкретной записи
func TestTUI_DataAccess_ViewSpecificData(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateView

	// Настраиваем мок для получения конкретной записи
	testEntry := &pb.DataEntry{
		Id:            "test-id",
		Name:          "Test Credentials",
		Description:   "Login data for example.com",
		Type:          pb.DataType_DATA_TYPE_CREDENTIALS,
		EncryptedData: []byte("encrypted-password"),
		Metadata:      "website: example.com",
		CreatedAt:     timestamppb.Now(),
	}
	mockClient.On("GetData", mock.Anything, "test-id").Return(testEntry, nil)

	// Симулируем загрузку записи
	msg := dataEntryLoadedMsg{entry: testEntry}
	newModel, _ := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, testEntry, updatedModel.viewingEntry)

	// Проверяем отображение деталей записи
	view := updatedModel.View()
	assert.Contains(t, view, "Test Credentials")
	assert.Contains(t, view, "Login data for example.com")
	assert.Contains(t, view, "test-id")
	assert.Contains(t, view, "Учетные данные")
	assert.Contains(t, view, "encrypted-password")
	assert.Contains(t, view, "website: example.com")
	assert.Contains(t, view, "Delete: удалить запись • Esc: назад")
}

// TestTUI_DataAccess_LoadDataEntry тестирует команду загрузки конкретной записи
func TestTUI_DataAccess_LoadDataEntry(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для получения данных
	testEntry := &pb.DataEntry{
		Id:            "test-id",
		Name:          "Test Entry",
		Description:   "Test Description",
		Type:          pb.DataType_DATA_TYPE_TEXT,
		EncryptedData: []byte("test-data"),
		Metadata:      "test-metadata",
	}
	mockClient.On("GetData", mock.Anything, "test-id").Return(testEntry, nil)

	cmd := model.loadDataEntry("test-id")
	assert.NotNil(t, cmd)

	// Выполняем команду
	msg := cmd()
	dataEntryMsg, ok := msg.(dataEntryLoadedMsg)
	assert.True(t, ok)
	assert.Equal(t, testEntry, dataEntryMsg.entry)

	mockClient.AssertExpectations(t)
}

// TestTUI_DataAccess_LoadDataList тестирует команду загрузки списка данных
func TestTUI_DataAccess_LoadDataList(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для загрузки списка
	entries := []*pb.DataEntry{
		{
			Id:          "1",
			Name:        "Entry 1",
			Description: "Description 1",
			Type:        pb.DataType_DATA_TYPE_CREDENTIALS,
		},
		{
			Id:          "2",
			Name:        "Entry 2",
			Description: "Description 2",
			Type:        pb.DataType_DATA_TYPE_TEXT,
		},
	}
	mockClient.On("ListData", mock.Anything, mock.Anything).Return(entries, nil)

	cmd := model.loadDataList()
	assert.NotNil(t, cmd)

	// Выполняем команду
	msg := cmd()
	dataListMsg, ok := msg.(dataListMsg)
	assert.True(t, ok)
	assert.Equal(t, entries, dataListMsg.entries)

	mockClient.AssertExpectations(t)
}

// TestTUI_DataAccess_DataTypes тестирует отображение различных типов данных
func TestTUI_DataAccess_DataTypes(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Тестируем все типы данных
	testCases := []struct {
		dataType pb.DataType
		expected string
	}{
		{pb.DataType_DATA_TYPE_CREDENTIALS, "Учетные данные"},
		{pb.DataType_DATA_TYPE_TEXT, "Текст"},
		{pb.DataType_DATA_TYPE_BINARY, "Бинарные данные"},
		{pb.DataType_DATA_TYPE_CARD, "Банковская карта"},
	}

	for _, tc := range testCases {
		result := model.getDataTypeString(tc.dataType)
		assert.Equal(t, tc.expected, result)
	}
}

// TestTUI_DataAccess_ViewEmptyList тестирует отображение пустого списка
func TestTUI_DataAccess_ViewEmptyList(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Симулируем пустой список
	msg := dataListMsg{entries: []*pb.DataEntry{}}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, 0, len(updatedModel.list.Items()))
	assert.Nil(t, cmd)
}

// TestTUI_DataAccess_ErrorHandling тестирует обработку ошибок при загрузке данных
func TestTUI_DataAccess_ErrorHandling(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для возврата ошибки
	mockClient.On("ListData", mock.Anything, mock.Anything).Return([]*pb.DataEntry{}, assert.AnError)

	cmd := model.loadDataList()
	assert.NotNil(t, cmd)

	// Выполняем команду
	msg := cmd()
	// Проверяем, что это ошибка
	assert.IsType(t, errorMsg{}, msg)
	errorMsg := msg.(errorMsg)
	assert.Contains(t, errorMsg.error, "ошибка загрузки данных")

	mockClient.AssertExpectations(t)
}

// TestTUI_DataAccess_DataEntryError тестирует обработку ошибок при загрузке конкретной записи
func TestTUI_DataAccess_DataEntryError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для возврата ошибки
	mockClient.On("GetData", mock.Anything, "test-id").Return((*pb.DataEntry)(nil), assert.AnError)

	cmd := model.loadDataEntry("test-id")
	assert.NotNil(t, cmd)

	// Выполняем команду
	msg := cmd()
	errorMsg, ok := msg.(errorMsg)
	assert.True(t, ok)
	assert.Contains(t, errorMsg.error, "ошибка загрузки записи")

	mockClient.AssertExpectations(t)
}

// TestTUI_DataAccess_ViewDataWithMetadata тестирует отображение данных с метаданными
func TestTUI_DataAccess_ViewDataWithMetadata(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateView

	// Создаем запись с метаданными
	testEntry := &pb.DataEntry{
		Id:            "test-id",
		Name:          "Bank Card",
		Description:   "Credit card information",
		Type:          pb.DataType_DATA_TYPE_CARD,
		EncryptedData: []byte("encrypted-card-data"),
		Metadata:      "bank: TestBank, card_type: Visa",
		CreatedAt:     timestamppb.Now(),
	}
	model.viewingEntry = testEntry

	// Проверяем отображение
	view := model.View()
	assert.Contains(t, view, "Bank Card")
	assert.Contains(t, view, "Credit card information")
	assert.Contains(t, view, "test-id")
	assert.Contains(t, view, "Банковская карта")
	assert.Contains(t, view, "encrypted-card-data")
	assert.Contains(t, view, "bank: TestBank, card_type: Visa")
}

// TestTUI_DataAccess_ViewDataWithoutMetadata тестирует отображение данных без метаданных
func TestTUI_DataAccess_ViewDataWithoutMetadata(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateView

	// Создаем запись без метаданных
	testEntry := &pb.DataEntry{
		Id:            "test-id",
		Name:          "Simple Note",
		Description:   "Just a note",
		Type:          pb.DataType_DATA_TYPE_TEXT,
		EncryptedData: []byte("simple-text"),
		Metadata:      "", // Пустые метаданные
		CreatedAt:     timestamppb.Now(),
	}
	model.viewingEntry = testEntry

	// Проверяем отображение
	view := model.View()
	assert.Contains(t, view, "Simple Note")
	assert.Contains(t, view, "Just a note")
	assert.Contains(t, view, "test-id")
	assert.Contains(t, view, "Текст")
	assert.Contains(t, view, "simple-text")
	// Метаданные не должны отображаться, если они пустые
	assert.NotContains(t, view, "Метаданные:")
}

// TestTUI_DataAccess_DataPrivacy тестирует приватность данных в TUI
func TestTUI_DataAccess_DataPrivacy(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	user1Entries := []*pb.DataEntry{
		{Id: "user1-entry-1", Name: "User1 Private Data", Description: "Very private", Type: pb.DataType_DATA_TYPE_CREDENTIALS, EncryptedData: []byte("user1-secret")},
	}
	user2Entries := []*pb.DataEntry{
		{Id: "user2-entry-1", Name: "User2 Private Data", Description: "Also private", Type: pb.DataType_DATA_TYPE_TEXT, EncryptedData: []byte("user2-secret")},
	}

	// Пользователь 1 загружает свои данные
	msg1 := dataListMsg{entries: user1Entries}
	newModel1, cmd1 := model.Update(msg1)
	updatedModel1 := newModel1.(*TUIModel)
	assert.Equal(t, 1, len(updatedModel1.list.Items()))
	assert.Nil(t, cmd1)

	// Пользователь 2 загружает свои данные
	msg2 := dataListMsg{entries: user2Entries}
	newModel2, cmd2 := model.Update(msg2)
	updatedModel2 := newModel2.(*TUIModel)
	assert.Equal(t, 1, len(updatedModel2.list.Items()))
	assert.Nil(t, cmd2)

}

// TestTUI_DataAccess_DataEncryption тестирует отображение зашифрованных данных
func TestTUI_DataAccess_DataEncryption(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateView

	// Создаем запись с зашифрованными данными
	secretData := "super-secret-password-123"
	testEntry := &pb.DataEntry{
		Id:            "test-id",
		Name:          "Encrypted Credentials",
		Description:   "Test encryption",
		Type:          pb.DataType_DATA_TYPE_CREDENTIALS,
		EncryptedData: []byte(secretData),
		Metadata:      "sensitive: true",
		CreatedAt:     timestamppb.Now(),
	}
	model.viewingEntry = testEntry

	// Проверяем, что данные отображаются в зашифрованном виде
	view := model.View()
	assert.Contains(t, view, "Encrypted Credentials")
	assert.Contains(t, view, "Test encryption")
	assert.Contains(t, view, "test-id")
	assert.Contains(t, view, "Учетные данные")
	assert.Contains(t, view, secretData)
	assert.Contains(t, view, "sensitive: true")
}
