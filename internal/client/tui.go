// Package client содержит TUI модель для интерактивного интерфейса.
package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	pb "github.com/GophKeeper/proto/gen/proto"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

// Состояния приложения
type appState int

const (
	stateLogin appState = iota
	stateRegister
	stateMain
	stateList
	stateCreate
	stateView
	stateOTP
)

// TUIModel представляет модель для TUI интерфейса.
type TUIModel struct {
	client ClientInterface
	logger *zap.Logger
	state  appState
	width  int
	height int
	err    error

	// Компоненты UI
	usernameInput textinput.Model
	passwordInput textinput.Model
	list          list.Model

	// Состояние
	currentUser   string
	message       string
	selectedEntry *listItem     // выбранная запись для просмотра
	viewingEntry  *pb.DataEntry // полная информация о просматриваемой записи

	// OTP состояние
	otpAccountInput textinput.Model
	otpSecret       string
	otpCode         string
	otpQRCodeURL    string
	otpBackupCodes  []string
	otpMessage      string

	// Состояние создания записи
	createNameInput        textinput.Model
	createDescriptionInput textinput.Model
	createDataInput        textinput.Model
	createTypeInput        textinput.Model
	createMetadataInput    textinput.Model
	createDataType         pb.DataType

	// Состояние синхронизации
	lastSyncTime time.Time
	syncMessage  string
	entriesCount int // Количество записей

	// Состояние загрузки
	isLoading      bool
	loadingMessage string
}

// Элемент списка
type listItem struct {
	title       string
	description string
	id          string
}

// Реализация интерфейса list.Item
func (i listItem) FilterValue() string { return i.title }
func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.description }

// ClientInterface определяет интерфейс для клиента
type ClientInterface interface {
	Login(ctx context.Context, username, password string) error
	Register(ctx context.Context, username, password string) error
	ListData(ctx context.Context, dataType *pb.DataType) ([]*pb.DataEntry, error)
	GetData(ctx context.Context, id string) (*pb.DataEntry, error)
	DeleteData(ctx context.Context, id string) error
	SyncData(ctx context.Context, lastSyncTime time.Time) (*pb.SyncDataResponse, error)
	CreateData(ctx context.Context, req *pb.CreateDataRequest) (*pb.DataEntry, error)
	CreateOTPSecret(ctx context.Context, issuer, accountName string) (*pb.CreateOTPSecretResponse, error)
	GenerateOTP(ctx context.Context, secret string) (*pb.GenerateOTPResponse, error)
	Close() error
}

// NewTUIModel создает новую модель TUI.
func NewTUIModel(client ClientInterface, logger *zap.Logger) *TUIModel {
	// Создаем поля ввода
	usernameInput := textinput.New()
	usernameInput.Placeholder = "Введите имя пользователя"
	usernameInput.Focus()
	usernameInput.CharLimit = 50
	usernameInput.Width = 30

	passwordInput := textinput.New()
	passwordInput.Placeholder = "Введите пароль"
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '•'
	passwordInput.CharLimit = 100
	passwordInput.Width = 30

	// Создаем список
	items := []list.Item{}
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "GophKeeper - Менеджер паролей"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	otpAccountInput := textinput.New()
	otpAccountInput.Placeholder = "Имя аккаунта (email/login)"
	otpAccountInput.CharLimit = 50
	otpAccountInput.Width = 30

	// Создаем поля для создания записи
	createNameInput := textinput.New()
	createNameInput.Placeholder = "Название записи"
	createNameInput.CharLimit = 100
	createNameInput.Width = 40

	createDescriptionInput := textinput.New()
	createDescriptionInput.Placeholder = "Описание (необязательно)"
	createDescriptionInput.CharLimit = 200
	createDescriptionInput.Width = 40

	createDataInput := textinput.New()
	createDataInput.Placeholder = "Данные (пароль, текст, и т.д.)"
	createDataInput.CharLimit = 500
	createDataInput.Width = 40

	createTypeInput := textinput.New()
	createTypeInput.Placeholder = "Тип данных (1-credentials, 2-text, 3-binary, 4-card)"
	createTypeInput.CharLimit = 10
	createTypeInput.Width = 40

	createMetadataInput := textinput.New()
	createMetadataInput.Placeholder = "Метаданные (необязательно)"
	createMetadataInput.CharLimit = 200
	createMetadataInput.Width = 40

	return &TUIModel{
		client:                 client,
		logger:                 logger,
		state:                  stateLogin,
		usernameInput:          usernameInput,
		passwordInput:          passwordInput,
		list:                   l,
		otpAccountInput:        otpAccountInput,
		createNameInput:        createNameInput,
		createDescriptionInput: createDescriptionInput,
		createDataInput:        createDataInput,
		createTypeInput:        createTypeInput,
		createMetadataInput:    createMetadataInput,
		createDataType:         pb.DataType_DATA_TYPE_CREDENTIALS, // По умолчанию
	}
}

// Init инициализирует модель.
func (m *TUIModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.tick(),
	)
}

// Update обновляет модель в ответ на сообщения.
func (m *TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)

	case tea.KeyMsg:
		switch m.state {
		case stateLogin:
			return m.updateLogin(msg)
		case stateRegister:
			return m.updateRegister(msg)
		case stateMain:
			return m.updateMain(msg)
		case stateList:
			return m.updateList(msg)
		case stateView:
			return m.updateView(msg)
		case stateOTP:
			return m.updateOTP(msg)
		case stateCreate:
			return m.updateCreate(msg)
		}

	case loginSuccessMsg:
		m.currentUser = msg.username
		m.state = stateMain
		m.message = ""
		m.isLoading = false
		m.entriesCount = 0 // Сбрасываем счетчик при входе
		// Синхронизируем данные при входе
		return m, m.syncData()

	case registerSuccessMsg:
		m.currentUser = msg.username
		m.state = stateMain
		m.message = ""
		m.isLoading = false
		m.entriesCount = 0 // Сбрасываем счетчик при регистрации
		// Синхронизируем данные при регистрации
		return m, m.syncData()

	case errorMsg:
		m.message = msg.error
		m.isLoading = false
		return m, nil

	case dataListMsg:
		// Конвертируем pb.DataEntry в listItem
		pbEntries, ok := msg.entries.([]*pb.DataEntry)
		if !ok {
			return m, nil
		}

		items := make([]list.Item, len(pbEntries))
		for i, entry := range pbEntries {
			items[i] = &listItem{
				title:       entry.Name,
				description: entry.Description,
				id:          entry.Id,
			}
		}
		m.list.SetItems(items)

		// Обновляем счетчик синхронизации
		m.entriesCount = len(pbEntries)
		if len(pbEntries) > 0 {
			m.syncMessage = fmt.Sprintf("Загружено %d записей", len(pbEntries))
		} else {
			m.syncMessage = "Записей не найдено"
		}
		return m, nil
	case entryDeletedMsg:
		m.state = stateList
		// Выполняем синхронизацию после удаления
		return m, tea.Batch(m.syncData(), m.loadDataList())
	case otpSecretMsg:
		m.otpSecret = msg.secret
		m.otpQRCodeURL = msg.qr
		m.otpBackupCodes = msg.backups
		m.otpCode = ""
		return m, nil
	case otpCodeMsg:
		m.otpCode = msg.code
		return m, nil
	case syncDataMsg:
		m.lastSyncTime = msg.lastSyncTime
		m.syncMessage = msg.message
		// Если в сообщении есть информация о записях, обновляем счетчик
		if strings.Contains(msg.message, "записей") {
			// Извлекаем количество из сообщения типа "Синхронизировано: * записей, * удалено"
			if strings.Contains(msg.message, "Синхронизировано:") {
				parts := strings.Split(msg.message, " ")
				if len(parts) >= 2 {
					if count, err := strconv.Atoi(parts[1]); err == nil {
						m.entriesCount = count
					}
				}
			}
		}
		// Если данные актуальны, продолжаем тикать
		if msg.message == "Данные актуальны" {
			return m, m.tick()
		}
		// Если есть изменения, загружаем актуальный список
		if strings.Contains(msg.message, "Синхронизировано:") {
			return m, m.loadDataList()
		}
		return m, nil
	case dataEntryMsg:
		m.selectedEntry = msg.entry
		// Загружаем полную информацию о записи
		return m, m.loadDataEntry(msg.entry.id)
	case dataEntryLoadedMsg:
		m.viewingEntry = msg.entry
		return m, nil
	case dataCreatedMsg:
		m.state = stateMain
		m.message = fmt.Sprintf("Запись '%s' успешно создана", msg.entry.Name)
		// Очищаем поля ввода
		m.createNameInput.SetValue("")
		m.createDescriptionInput.SetValue("")
		m.createDataInput.SetValue("")
		m.createTypeInput.SetValue("")
		m.createMetadataInput.SetValue("")
		// Сначала загружаем список, потом синхронизируемся
		return m, m.loadDataList()

	case tickMsg:
		// Автоматическая синхронизация каждые 5 секунд
		if m.currentUser != "" {
			return m, tea.Batch(m.syncData(), m.tick())
		}
		return m, m.tick()
	}

	return m, cmd
}

// View отображает модель.
func (m *TUIModel) View() string {
	switch m.state {
	case stateLogin:
		return m.viewLogin()
	case stateRegister:
		return m.viewRegister()
	case stateMain:
		return m.viewMain()
	case stateList:
		return m.viewList()
	case stateView:
		return m.viewView()
	case stateOTP:
		return m.viewOTP()
	case stateCreate:
		return m.viewCreate()
	default:
		return "Неизвестное состояние"
	}
}

// updateLogin обновляет состояние входа.
func (m *TUIModel) updateLogin(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.isLoading = true
		m.loadingMessage = "Выход из системы..."
		return m, tea.Quit

	case tea.KeyTab, tea.KeyShiftTab, tea.KeyEnter, tea.KeyUp, tea.KeyDown:
		if m.usernameInput.Focused() {
			m.usernameInput.Blur()
			m.passwordInput.Focus()
		} else {
			m.passwordInput.Blur()
			m.usernameInput.Focus()
		}

	case tea.KeyCtrlS:
		// Вход в систему
		m.isLoading = true
		m.loadingMessage = "Вход в систему..."
		return m, m.login()

	case tea.KeyCtrlR:
		// Переход к регистрации
		m.state = stateRegister
		m.message = ""
		return m, nil
	}

	// Обновляем активное поле ввода
	if m.usernameInput.Focused() {
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	} else {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}

	return m, cmd
}

// updateRegister обновляет состояние регистрации.
func (m *TUIModel) updateRegister(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.isLoading = true
		m.loadingMessage = "Выход из системы..."
		return m, tea.Quit

	case tea.KeyTab, tea.KeyShiftTab, tea.KeyEnter, tea.KeyUp, tea.KeyDown:
		if m.usernameInput.Focused() {
			m.usernameInput.Blur()
			m.passwordInput.Focus()
		} else {
			m.passwordInput.Blur()
			m.usernameInput.Focus()
		}

	case tea.KeyCtrlS:
		// Регистрация
		m.isLoading = true
		m.loadingMessage = "Регистрация..."
		return m, m.register()

	case tea.KeyCtrlL:
		// Переход ко входу
		m.state = stateLogin
		m.message = ""
		return m, nil
	}

	// Обновляем активное поле ввода
	if m.usernameInput.Focused() {
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	} else {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}

	return m, cmd
}

// updateMain обновляет главное состояние.
func (m *TUIModel) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.isLoading = true
		m.loadingMessage = "Выход из системы..."
		return m, tea.Quit

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "1":
			m.state = stateList
			// Синхронизируем данные перед загрузкой списка
			return m, tea.Batch(m.syncData(), m.loadDataList())
		case "2":
			m.state = stateCreate
			// Устанавливаем фокус на первое поле
			m.createNameInput.Focus()
			return m, nil
		case "3":
			m.state = stateOTP
			m.otpAccountInput.Focus()
			return m, nil
		case "s":
			// Ручная синхронизация данных
			return m, m.syncData()
		case "q":
			m.isLoading = true
			m.loadingMessage = "Выход из системы..."
			return m, tea.Quit
		}
	}

	return m, nil
}

// updateList обновляет состояние списка.
func (m *TUIModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.state = stateMain
		return m, nil
	case tea.KeyEnter:
		// Пользователь выбрал запись для просмотра
		if len(m.list.Items()) > 0 {
			selectedItem := m.list.SelectedItem()
			if item, ok := selectedItem.(*listItem); ok {
				m.selectedEntry = item
				m.state = stateView
				return m, m.loadDataEntry(item.id)
			}
		}
	case tea.KeyUp, tea.KeyDown:
		// Передаем стрелки в список для правильной навигации
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	// Для остальных клавиш обновляем список
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// updateView обновляет состояние просмотра записи.
func (m *TUIModel) updateView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.state = stateList
		return m, nil
	case tea.KeyDelete, tea.KeyBackspace:
		if m.selectedEntry != nil {
			return m, m.deleteEntry(m.selectedEntry.id)
		}
	}
	return m, nil
}

// updateCreate обновляет состояние создания записи.
func (m *TUIModel) updateCreate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.state = stateMain
		// Сбрасываем фокус
		m.createNameInput.Blur()
		m.createDescriptionInput.Blur()
		m.createDataInput.Blur()
		m.createTypeInput.Blur()
		m.createMetadataInput.Blur()
		return m, nil

	case tea.KeyTab, tea.KeyShiftTab, tea.KeyEnter, tea.KeyUp, tea.KeyDown:
		// Переключаем фокус между полями
		if m.createNameInput.Focused() {
			m.createNameInput.Blur()
			m.createDescriptionInput.Focus()
		} else if m.createDescriptionInput.Focused() {
			m.createDescriptionInput.Blur()
			m.createDataInput.Focus()
		} else if m.createDataInput.Focused() {
			m.createDataInput.Blur()
			m.createTypeInput.Focus()
		} else if m.createTypeInput.Focused() {
			m.createTypeInput.Blur()
			m.createMetadataInput.Focus()
		} else {
			m.createMetadataInput.Blur()
			m.createNameInput.Focus()
		}

	case tea.KeyCtrlS:
		// Создание записи
		return m, m.createDataEntry()
	}

	// Обновляем активное поле ввода
	if m.createNameInput.Focused() {
		m.createNameInput, cmd = m.createNameInput.Update(msg)
	} else if m.createDescriptionInput.Focused() {
		m.createDescriptionInput, cmd = m.createDescriptionInput.Update(msg)
	} else if m.createDataInput.Focused() {
		m.createDataInput, cmd = m.createDataInput.Update(msg)
	} else if m.createTypeInput.Focused() {
		m.createTypeInput, cmd = m.createTypeInput.Update(msg)
	} else if m.createMetadataInput.Focused() {
		m.createMetadataInput, cmd = m.createMetadataInput.Update(msg)
	}

	return m, cmd
}

// updateOTP обновляет состояние генерации OTP.
func (m *TUIModel) updateOTP(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.state = stateMain
		return m, nil
	case tea.KeyCtrlS:
		return m, m.createOTPSecret()
	case tea.KeyCtrlG:
		if m.otpSecret != "" {
			return m, m.generateOTPCode()
		}
	}
	m.otpAccountInput, _ = m.otpAccountInput.Update(msg)
	return m, nil
}

func (m *TUIModel) createOTPSecret() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.client.CreateOTPSecret(ctx, "GophKeeper", m.otpAccountInput.Value())
		if err != nil {
			return errorMsg{error: fmt.Sprintf("ошибка создания OTP секрета: %v", err)}
		}
		return otpSecretMsg{secret: resp.Secret, qr: resp.QrCodeUrl, backups: resp.BackupCodes}
	}
}

func (m *TUIModel) generateOTPCode() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.client.GenerateOTP(ctx, m.otpSecret)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("ошибка генерации OTP: %v", err)}
		}
		return otpCodeMsg{code: resp.Code}
	}
}

// viewLogin отображает экран входа.
func (m *TUIModel) viewLogin() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔐 GophKeeper - Вход в систему"))
	b.WriteString("\n\n")

	if m.isLoading {
		b.WriteString(helpStyle.Render("⏳ " + m.loadingMessage))
		b.WriteString("\n\n")
	} else if m.message != "" {
		b.WriteString(errorStyle.Render(m.message))
		b.WriteString("\n\n")
	}

	b.WriteString("Имя пользователя:\n")
	b.WriteString(m.usernameInput.View())
	b.WriteString("\n\n")

	b.WriteString("Пароль:\n")
	b.WriteString(m.passwordInput.View())
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("Ctrl+S: Войти • Ctrl+R: Регистрация • Ctrl+C: Выход"))

	return containerStyle.Render(b.String())
}

// viewRegister отображает экран регистрации.
func (m *TUIModel) viewRegister() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📝 GophKeeper - Регистрация"))
	b.WriteString("\n\n")

	if m.isLoading {
		b.WriteString(helpStyle.Render("⏳ " + m.loadingMessage))
		b.WriteString("\n\n")
	} else if m.message != "" {
		b.WriteString(errorStyle.Render(m.message))
		b.WriteString("\n\n")
	}

	b.WriteString("Имя пользователя:\n")
	b.WriteString(m.usernameInput.View())
	b.WriteString("\n\n")

	b.WriteString("Пароль:\n")
	b.WriteString(m.passwordInput.View())
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("Ctrl+S: Зарегистрироваться • Ctrl+L: Вход • Ctrl+C: Выход"))

	return containerStyle.Render(b.String())
}

// viewMain отображает главное меню.
func (m *TUIModel) viewMain() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("🏠 GophKeeper - Добро пожаловать, %s!", m.currentUser)))
	b.WriteString("\n\n")

	// Показываем информацию о синхронизации
	if !m.lastSyncTime.IsZero() {
		b.WriteString(fmt.Sprintf("🔄 Последняя синхронизация: %s\n", m.lastSyncTime.Format("15:04:05")))
	}

	// Показываем информацию о записях
	if m.syncMessage != "" && m.syncMessage != "Данные актуальны" {
		b.WriteString(fmt.Sprintf("📊 %s\n", m.syncMessage))
	} else if m.entriesCount > 0 {
		b.WriteString(fmt.Sprintf("📊 Загружено %d записей\n", m.entriesCount))
	} else if m.syncMessage == "Данные актуальны" {
		b.WriteString("📊 Данные актуальны\n")
	}
	b.WriteString("\n")

	b.WriteString("Выберите действие:\n\n")
	b.WriteString("1. 📋 Просмотр данных\n")
	b.WriteString("2. ➕ Добавить данные\n")
	b.WriteString("3. 🔑 Генератор OTP\n")
	b.WriteString("s. 🔄 Синхронизировать данные\n")
	b.WriteString("q. ❌ Выход\n\n")

	b.WriteString(helpStyle.Render("Используйте цифры для выбора • s: Синхронизация • q: Выход"))

	return containerStyle.Render(b.String())
}

// viewList отображает список данных.
func (m *TUIModel) viewList() string {
	var b strings.Builder
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Enter: просмотр записи • Esc: назад"))
	return b.String()
}

// viewOTP отображает экран генерации OTP.
func (m *TUIModel) viewView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📋 Просмотр записи"))
	b.WriteString("\n\n")

	if m.viewingEntry != nil {
		b.WriteString("Название: " + m.viewingEntry.Name + "\n")
		b.WriteString("Описание: " + m.viewingEntry.Description + "\n")
		b.WriteString("ID: " + m.viewingEntry.Id + "\n")
		b.WriteString("Тип: " + m.getDataTypeString(m.viewingEntry.Type) + "\n")
		b.WriteString("Данные: " + string(m.viewingEntry.EncryptedData) + "\n")
		if m.viewingEntry.Metadata != "" {
			b.WriteString("Метаданные: " + m.viewingEntry.Metadata + "\n")
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Delete: удалить запись • Esc: назад"))
	} else {
		b.WriteString("Запись не найдена\n")
		b.WriteString(helpStyle.Render("Esc: назад"))
	}

	return containerStyle.Render(b.String())
}

func (m *TUIModel) viewCreate() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("➕ Добавление записи"))
	b.WriteString("\n\n")

	// Показываем поля ввода
	b.WriteString("Название записи:\n")
	b.WriteString(m.createNameInput.View())
	b.WriteString("\n\n")

	b.WriteString("Описание (необязательно):\n")
	b.WriteString(m.createDescriptionInput.View())
	b.WriteString("\n\n")

	b.WriteString("Данные (пароль, текст, и т.д.):\n")
	b.WriteString(m.createDataInput.View())
	b.WriteString("\n\n")

	b.WriteString("Тип данных (1-credentials, 2-text, 3-binary, 4-card):\n")
	b.WriteString(m.createTypeInput.View())
	b.WriteString("\n\n")

	b.WriteString("Метаданные (необязательно):\n")
	b.WriteString(m.createMetadataInput.View())
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("Tab: переключение полей • Ctrl+S: создать запись • Esc: назад"))
	return containerStyle.Render(b.String())
}

func (m *TUIModel) viewOTP() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔑 Генерация OTP"))
	b.WriteString("\n\n")
	b.WriteString("Имя аккаунта (email/login):\n")
	b.WriteString(m.otpAccountInput.View() + "\n\n")
	if m.otpSecret != "" {
		b.WriteString("Секрет: " + m.otpSecret + "\n")
		b.WriteString("QR: " + m.otpQRCodeURL + "\n")
		b.WriteString("Резервные коды:\n")
		for _, code := range m.otpBackupCodes {
			b.WriteString("  " + code + "\n")
		}
		if m.otpCode != "" {
			b.WriteString("\nТекущий OTP: " + m.otpCode + "\n")
		}
		b.WriteString(helpStyle.Render("Ctrl+G: сгенерировать OTP • Esc: назад"))
	} else {
		b.WriteString(helpStyle.Render("Ctrl+S: создать секрет • Esc: назад"))
	}
	return containerStyle.Render(b.String())
}

// getDataTypeString возвращает строковое представление типа данных
func (m *TUIModel) getDataTypeString(dataType pb.DataType) string {
	switch dataType {
	case pb.DataType_DATA_TYPE_CREDENTIALS:
		return "Учетные данные"
	case pb.DataType_DATA_TYPE_TEXT:
		return "Текст"
	case pb.DataType_DATA_TYPE_BINARY:
		return "Бинарные данные"
	case pb.DataType_DATA_TYPE_CARD:
		return "Банковская карта"
	default:
		return "Неизвестный тип"
	}
}

// Команды для асинхронных операций
func (m *TUIModel) login() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.client.Login(ctx, m.usernameInput.Value(), m.passwordInput.Value())
		if err != nil {
			return errorMsg{error: fmt.Sprintf("ошибка входа: %v", err)}
		}
		return loginSuccessMsg{username: m.usernameInput.Value()}
	}
}

func (m *TUIModel) register() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.client.Register(ctx, m.usernameInput.Value(), m.passwordInput.Value())
		if err != nil {
			return errorMsg{error: fmt.Sprintf("ошибка регистрации: %v", err)}
		}
		return registerSuccessMsg{username: m.usernameInput.Value()}
	}
}

func (m *TUIModel) loadDataList() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		entries, err := m.client.ListData(ctx, nil)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("ошибка загрузки данных: %v", err)}
		}
		return dataListMsg{entries: entries}
	}
}

func (m *TUIModel) deleteEntry(id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.client.DeleteData(ctx, id)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("ошибка удаления: %v", err)}
		}
		return entryDeletedMsg{}
	}
}

func (m *TUIModel) syncData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.client.SyncData(ctx, m.lastSyncTime)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("ошибка синхронизации: %v", err)}
		}

		// Обновляем время последней синхронизации
		lastSyncTime := time.Now()
		if resp.LastSyncTime != nil {
			lastSyncTime = resp.LastSyncTime.AsTime()
		}

		var message string
		if len(resp.DataEntries) > 0 || len(resp.DeletedIds) > 0 {
			message = fmt.Sprintf("Синхронизировано: %d записей, %d удалено",
				len(resp.DataEntries), len(resp.DeletedIds))
		} else {
			message = "Данные актуальны"
		}

		return syncDataMsg{
			lastSyncTime: lastSyncTime,
			message:      message,
		}
	}
}

func (m *TUIModel) loadDataEntry(id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		entry, err := m.client.GetData(ctx, id)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("ошибка загрузки записи: %v", err)}
		}

		return dataEntryLoadedMsg{entry: entry}
	}
}

func (m *TUIModel) tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *TUIModel) createDataEntry() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Проверяем обязательные поля
		if m.createNameInput.Value() == "" {
			return errorMsg{error: "Название записи обязательно"}
		}
		if m.createDataInput.Value() == "" {
			return errorMsg{error: "Данные записи обязательны"}
		}

		// Определяем тип данных
		dataType := m.createDataType
		if m.createTypeInput.Value() != "" {
			switch m.createTypeInput.Value() {
			case "1":
				dataType = pb.DataType_DATA_TYPE_CREDENTIALS
			case "2":
				dataType = pb.DataType_DATA_TYPE_TEXT
			case "3":
				dataType = pb.DataType_DATA_TYPE_BINARY
			case "4":
				dataType = pb.DataType_DATA_TYPE_CARD
			default:
				return errorMsg{error: "Неверный тип данных. Используйте 1-4"}
			}
		}

		// Создаем запрос
		req := &pb.CreateDataRequest{
			Type:          dataType,
			Name:          m.createNameInput.Value(),
			Description:   m.createDescriptionInput.Value(),
			EncryptedData: []byte(m.createDataInput.Value()), // TODO: Зашифровать данные
			Metadata:      m.createMetadataInput.Value(),
		}

		// Отправляем запрос
		entry, err := m.client.CreateData(ctx, req)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("ошибка создания записи: %v", err)}
		}

		return dataCreatedMsg{entry: entry}
	}
}

// Сообщения
type loginSuccessMsg struct{ username string }
type registerSuccessMsg struct{ username string }
type dataListMsg struct{ entries interface{} }
type dataEntryMsg struct{ entry *listItem }
type dataEntryLoadedMsg struct{ entry *pb.DataEntry }
type entryDeletedMsg struct{}
type dataCreatedMsg struct{ entry *pb.DataEntry }
type otpSecretMsg struct {
	secret  string
	qr      string
	backups []string
}
type otpCodeMsg struct{ code string }
type syncDataMsg struct {
	lastSyncTime time.Time
	message      string
}
type errorMsg struct{ error string }
type tickMsg struct{}

// Стили
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	containerStyle = lipgloss.NewStyle().
			Padding(1, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	paginationStyle = list.DefaultStyles().PaginationStyle.
			PaddingLeft(4)
)
