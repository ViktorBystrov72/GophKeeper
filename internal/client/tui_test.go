package client

import (
	"context"
	"testing"
	"time"

	pb "github.com/GophKeeper/proto/gen/proto"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MockClient - мок для тестирования
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Login(ctx context.Context, username, password string) error {
	args := m.Called(ctx, username, password)
	return args.Error(0)
}

func (m *MockClient) Register(ctx context.Context, username, password string) error {
	args := m.Called(ctx, username, password)
	return args.Error(0)
}

func (m *MockClient) CreateData(ctx context.Context, req *pb.CreateDataRequest) (*pb.DataEntry, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.DataEntry), args.Error(1)
}

func (m *MockClient) ListData(ctx context.Context, dataType *pb.DataType) ([]*pb.DataEntry, error) {
	args := m.Called(ctx, dataType)
	return args.Get(0).([]*pb.DataEntry), args.Error(1)
}

func (m *MockClient) UpdateData(ctx context.Context, req *pb.UpdateDataRequest) (*pb.DataEntry, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.DataEntry), args.Error(1)
}

func (m *MockClient) DeleteData(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockClient) CreateOTPSecret(ctx context.Context, issuer, accountName string) (*pb.CreateOTPSecretResponse, error) {
	args := m.Called(ctx, issuer, accountName)
	return args.Get(0).(*pb.CreateOTPSecretResponse), args.Error(1)
}

func (m *MockClient) GenerateOTP(ctx context.Context, secret string) (*pb.GenerateOTPResponse, error) {
	args := m.Called(ctx, secret)
	return args.Get(0).(*pb.GenerateOTPResponse), args.Error(1)
}

func (m *MockClient) SyncData(ctx context.Context, lastSyncTime time.Time) (*pb.SyncDataResponse, error) {
	args := m.Called(ctx, lastSyncTime)
	return args.Get(0).(*pb.SyncDataResponse), args.Error(1)
}

func (m *MockClient) GetData(ctx context.Context, id string) (*pb.DataEntry, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*pb.DataEntry), args.Error(1)
}

func (m *MockClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewTUIModel(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}

	model := NewTUIModel(mockClient, logger)

	assert.NotNil(t, model)
	assert.Equal(t, stateLogin, model.state)
	assert.NotNil(t, model.usernameInput)
	assert.NotNil(t, model.passwordInput)
	assert.NotNil(t, model.list)
	assert.NotNil(t, model.otpAccountInput)
	assert.Equal(t, mockClient, model.client)
	assert.Equal(t, logger, model.logger)
}

func TestTUIModel_Init(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestTUIModel_Update_WindowSize(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Тестируем обработку изменения размера окна
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, cmd := model.Update(msg)

	assert.Equal(t, 100, newModel.(*TUIModel).width)
	assert.Equal(t, 50, newModel.(*TUIModel).height)
	assert.Nil(t, cmd)
}

func TestTUIModel_Update_LoginSuccess(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Тестируем успешный вход
	msg := loginSuccessMsg{username: "testuser"}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, "testuser", updatedModel.currentUser)
	assert.Equal(t, stateMain, updatedModel.state)
	assert.Equal(t, "", updatedModel.message)
	assert.NotNil(t, cmd) // Теперь возвращается команда синхронизации
}

func TestTUIModel_Update_RegisterSuccess(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Тестируем успешную регистрацию
	msg := registerSuccessMsg{username: "newuser"}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, "newuser", updatedModel.currentUser)
	assert.Equal(t, stateMain, updatedModel.state)
	assert.Equal(t, "", updatedModel.message)
	assert.NotNil(t, cmd) // Теперь возвращается команда синхронизации
}

func TestTUIModel_Update_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Тестируем обработку ошибки
	msg := errorMsg{error: "test error"}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, "test error", updatedModel.message)
	assert.Nil(t, cmd)
}

func TestTUIModel_UpdateLogin_CtrlS(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Устанавливаем значения в полях ввода
	model.usernameInput.SetValue("testuser")
	model.passwordInput.SetValue("testpass")

	// Симулируем нажатие Ctrl+S
	msg := tea.KeyMsg{Type: tea.KeyCtrlS}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)
}

func TestTUIModel_UpdateLogin_CtrlR(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Симулируем нажатие Ctrl+R для перехода к регистрации
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, stateRegister, updatedModel.state)
	assert.Equal(t, "", updatedModel.message)
	assert.Nil(t, cmd)
}

func TestTUIModel_UpdateLogin_Tab(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Изначально фокус на поле username
	assert.True(t, model.usernameInput.Focused())

	// Симулируем нажатие Tab
	msg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.False(t, updatedModel.usernameInput.Focused())
	assert.True(t, updatedModel.passwordInput.Focused())
	assert.Nil(t, cmd)
}

func TestTUIModel_UpdateRegister_CtrlS(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateRegister

	// Устанавливаем значения в полях ввода
	model.usernameInput.SetValue("newuser")
	model.passwordInput.SetValue("newpass")

	// Симулируем нажатие Ctrl+S
	msg := tea.KeyMsg{Type: tea.KeyCtrlS}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)
	// Команда выполняется асинхронно, поэтому мок не вызывается сразу
}

func TestTUIModel_UpdateRegister_CtrlL(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateRegister

	// Симулируем нажатие Ctrl+L для возврата к входу
	msg := tea.KeyMsg{Type: tea.KeyCtrlL}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, stateLogin, updatedModel.state)
	assert.Equal(t, "", updatedModel.message)
	assert.Nil(t, cmd)
}

func TestTUIModel_UpdateMain_Key1(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateMain

	// Симулируем нажатие клавиши "1"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, stateList, updatedModel.state)
	assert.NotNil(t, cmd) // loadDataList команда
}

func TestTUIModel_UpdateMain_Key2(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateMain

	// Симулируем нажатие клавиши "2"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, stateCreate, updatedModel.state)
	assert.Nil(t, cmd)
}

func TestTUIModel_UpdateMain_Key3(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateMain

	// Симулируем нажатие клавиши "3"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, stateOTP, updatedModel.state)
	assert.Nil(t, cmd)
}

func TestTUIModel_UpdateMain_KeyQ(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateMain

	// Симулируем нажатие клавиши "q"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)

	// Проверяем, что команда не nil (это tea.Quit)
	assert.NotNil(t, cmd)
}

func TestTUIModel_UpdateOTP_CtrlS(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateOTP

	// Устанавливаем значение в поле ввода
	model.otpAccountInput.SetValue("test@example.com")

	// Симулируем нажатие Ctrl+S
	msg := tea.KeyMsg{Type: tea.KeyCtrlS}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)
}

func TestTUIModel_UpdateOTP_CtrlG(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateOTP
	model.otpSecret = "testsecret"

	// Симулируем нажатие Ctrl+G
	msg := tea.KeyMsg{Type: tea.KeyCtrlG}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)
}

func TestTUIModel_View_Login(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateLogin

	view := model.View()
	assert.Contains(t, view, "🔐 GophKeeper - Вход в систему")
	assert.Contains(t, view, "Имя пользователя:")
	assert.Contains(t, view, "Пароль:")
	assert.Contains(t, view, "Ctrl+S: Войти")
}

func TestTUIModel_View_Register(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateRegister

	view := model.View()
	assert.Contains(t, view, "📝 GophKeeper - Регистрация")
	assert.Contains(t, view, "Имя пользователя:")
	assert.Contains(t, view, "Пароль:")
	assert.Contains(t, view, "Ctrl+S: Зарегистрироваться")
}

func TestTUIModel_View_Main(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateMain
	model.currentUser = "testuser"

	view := model.View()
	assert.Contains(t, view, "🏠 GophKeeper - Добро пожаловать, testuser!")
	assert.Contains(t, view, "1. 📋 Просмотр данных")
	assert.Contains(t, view, "2. ➕ Добавить данные")
	assert.Contains(t, view, "3. 🔑 Генератор OTP")
	assert.Contains(t, view, "q. ❌ Выход")
}

func TestTUIModel_View_OTP(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateOTP

	view := model.View()
	assert.Contains(t, view, "🔑 Генерация OTP")
	assert.Contains(t, view, "Имя аккаунта (email/login):")
	assert.Contains(t, view, "Ctrl+S: создать секрет")
}

func TestTUIModel_View_OTP_WithSecret(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateOTP
	model.otpSecret = "testsecret"
	model.otpQRCodeURL = "testqr"
	model.otpBackupCodes = []string{"12345678", "87654321"}
	model.otpCode = "123456"

	view := model.View()
	assert.Contains(t, view, "Секрет: testsecret")
	assert.Contains(t, view, "QR: testqr")
	assert.Contains(t, view, "12345678")
	assert.Contains(t, view, "87654321")
	assert.Contains(t, view, "Текущий OTP: 123456")
	assert.Contains(t, view, "Ctrl+G: сгенерировать OTP")
}

func TestTUIModel_Update_DataList(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Создаем тестовые данные в формате pb.DataEntry
	entries := []*pb.DataEntry{
		{Id: "1", Name: "Test Entry 1", Description: "Description 1"},
		{Id: "2", Name: "Test Entry 2", Description: "Description 2"},
	}

	msg := dataListMsg{entries: entries}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, 2, len(updatedModel.list.Items()))
	assert.Nil(t, cmd)
}

func TestTUIModel_Update_EntryDeleted(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateView

	msg := entryDeletedMsg{}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, stateList, updatedModel.state)
	assert.NotNil(t, cmd) // loadDataList команда
}

func TestTUIModel_Update_OTPSecret(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateOTP

	msg := otpSecretMsg{
		secret:  "testsecret",
		qr:      "testqr",
		backups: []string{"12345678", "87654321"},
	}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, "testsecret", updatedModel.otpSecret)
	assert.Equal(t, "testqr", updatedModel.otpQRCodeURL)
	assert.Equal(t, []string{"12345678", "87654321"}, updatedModel.otpBackupCodes)
	assert.Equal(t, "", updatedModel.otpCode)
	assert.Nil(t, cmd)
}

func TestTUIModel_Update_OTPCode(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.state = stateOTP

	msg := otpCodeMsg{code: "123456"}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, "123456", updatedModel.otpCode)
	assert.Nil(t, cmd)
}

func TestTUIModel_Update_SyncData(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Тестируем обработку сообщения синхронизации
	lastSyncTime := time.Now()
	msg := syncDataMsg{
		lastSyncTime: lastSyncTime,
		message:      "Синхронизировано: 5 записей, 2 удалено",
	}
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*TUIModel)
	assert.Equal(t, lastSyncTime, updatedModel.lastSyncTime)
	assert.Equal(t, "Синхронизировано: 5 записей, 2 удалено", updatedModel.syncMessage)
	// Команда может быть не nil, если есть изменения
	assert.NotNil(t, cmd)
}

func TestListItem_Interface(t *testing.T) {
	item := listItem{
		title:       "Test Title",
		description: "Test Description",
		id:          "test-id",
	}

	assert.Equal(t, "Test Title", item.FilterValue())
	assert.Equal(t, "Test Title", item.Title())
	assert.Equal(t, "Test Description", item.Description())
}

func TestTUIModel_Login_Command(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для успешного входа
	mockClient.On("Login", mock.Anything, "testuser", "testpass").Return(nil)

	// Устанавливаем значения в полях ввода
	model.usernameInput.SetValue("testuser")
	model.passwordInput.SetValue("testpass")

	cmd := model.login()
	assert.NotNil(t, cmd)

	// Выполняем команду
	msg := cmd()
	loginMsg, ok := msg.(loginSuccessMsg)
	assert.True(t, ok)
	assert.Equal(t, "testuser", loginMsg.username)

	mockClient.AssertExpectations(t)
}

func TestTUIModel_Register_Command(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для успешной регистрации
	mockClient.On("Register", mock.Anything, "newuser", "newpass").Return(nil)

	// Устанавливаем значения в полях ввода
	model.usernameInput.SetValue("newuser")
	model.passwordInput.SetValue("newpass")

	cmd := model.register()
	assert.NotNil(t, cmd)

	// Выполняем команду
	msg := cmd()
	registerMsg, ok := msg.(registerSuccessMsg)
	assert.True(t, ok)
	assert.Equal(t, "newuser", registerMsg.username)

	mockClient.AssertExpectations(t)
}

func TestTUIModel_LoadDataList_Command(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для загрузки данных
	entries := []*pb.DataEntry{
		{
			Id:          "1",
			Name:        "Test Entry",
			Description: "Test Description",
			Type:        pb.DataType_DATA_TYPE_CREDENTIALS,
			CreatedAt:   timestamppb.Now(),
		},
	}
	mockClient.On("ListData", mock.Anything, mock.Anything).Return(entries, nil)

	cmd := model.loadDataList()
	assert.NotNil(t, cmd)

	// Выполняем команду
	msg := cmd()
	dataMsg, ok := msg.(dataListMsg)
	assert.True(t, ok)
	assert.Equal(t, entries, dataMsg.entries)

	mockClient.AssertExpectations(t)
}

func TestTUIModel_DeleteEntry_Command(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для удаления записи
	mockClient.On("DeleteData", mock.Anything, "test-id").Return(nil)

	cmd := model.deleteEntry("test-id")
	assert.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(entryDeletedMsg)
	assert.True(t, ok)

	mockClient.AssertExpectations(t)
}

func TestTUIModel_CreateOTPSecret_Command(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для создания OTP секрета
	otpResponse := &pb.CreateOTPSecretResponse{
		Secret:      "testsecret",
		QrCodeUrl:   "testqr",
		BackupCodes: []string{"12345678", "87654321"},
	}
	mockClient.On("CreateOTPSecret", mock.Anything, "GophKeeper", "test@example.com").Return(otpResponse, nil)

	// Устанавливаем значение в поле ввода
	model.otpAccountInput.SetValue("test@example.com")

	cmd := model.createOTPSecret()
	assert.NotNil(t, cmd)

	msg := cmd()
	otpSecretMsg, ok := msg.(otpSecretMsg)
	assert.True(t, ok)
	assert.Equal(t, "testsecret", otpSecretMsg.secret)
	assert.Equal(t, "testqr", otpSecretMsg.qr)
	assert.Equal(t, []string{"12345678", "87654321"}, otpSecretMsg.backups)

	mockClient.AssertExpectations(t)
}

func TestTUIModel_GenerateOTPCode_Command(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для генерации OTP кода
	otpResponse := &pb.GenerateOTPResponse{Code: "123456"}
	mockClient.On("GenerateOTP", mock.Anything, "testsecret").Return(otpResponse, nil)

	model.otpSecret = "testsecret"

	cmd := model.generateOTPCode()
	assert.NotNil(t, cmd)

	msg := cmd()
	otpCodeMsg, ok := msg.(otpCodeMsg)
	assert.True(t, ok)
	assert.Equal(t, "123456", otpCodeMsg.code)

	mockClient.AssertExpectations(t)
}

func TestTUIModel_SyncData_Command(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)
	model.lastSyncTime = time.Now().Add(-time.Hour)

	// Настраиваем мок для синхронизации
	syncResponse := &pb.SyncDataResponse{
		DataEntries: []*pb.DataEntry{
			{Id: "1", Name: "Test Entry"},
		},
		DeletedIds:   []string{"deleted1"},
		LastSyncTime: timestamppb.Now(),
	}
	mockClient.On("SyncData", mock.Anything, model.lastSyncTime).Return(syncResponse, nil)

	cmd := model.syncData()
	assert.NotNil(t, cmd)

	msg := cmd()
	syncMsg, ok := msg.(syncDataMsg)
	assert.True(t, ok)
	assert.Contains(t, syncMsg.message, "Синхронизировано: 1 записей, 1 удалено")

	mockClient.AssertExpectations(t)
}

func TestTUIModel_LoadDataEntry_Command(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockClient := &MockClient{}
	model := NewTUIModel(mockClient, logger)

	// Настраиваем мок для получения данных
	testEntry := &pb.DataEntry{
		Id:          "test-id",
		Name:        "Test Entry",
		Description: "Test Description",
		Type:        pb.DataType_DATA_TYPE_CREDENTIALS,
	}
	mockClient.On("GetData", mock.Anything, "test-id").Return(testEntry, nil)

	cmd := model.loadDataEntry("test-id")
	assert.NotNil(t, cmd)

	msg := cmd()
	dataEntryLoadedMsg, ok := msg.(dataEntryLoadedMsg)
	assert.True(t, ok)
	assert.Equal(t, testEntry, dataEntryLoadedMsg.entry)

	mockClient.AssertExpectations(t)
}
