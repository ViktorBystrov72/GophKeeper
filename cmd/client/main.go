// Package main инициализирует и запускает клиент GophKeeper.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/GophKeeper/internal/client"
	"github.com/GophKeeper/internal/config"
	"github.com/GophKeeper/internal/logger"
	"github.com/GophKeeper/internal/templates"
	"github.com/GophKeeper/internal/version"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

func main() {
	if shouldShowVersionOrHelp() {
		return
	}
	if err := runApplication(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

// shouldShowVersionOrHelp проверяет, нужно ли показать версию или справку
func shouldShowVersionOrHelp() bool {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			// Устанавливаем значения по умолчанию для версии
			version.SetDefaults()
			version.PrintVersionInfo("GophKeeper Client")
			return true
		case "help", "--help", "-h":
			showHelp()
			return true
		}
	}
	return false
}

// runApplication выполняет основную логику приложения
func runApplication() error {
	ctx := context.Background()

	// Загружаем конфигурацию
	cfg, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Инициализируем логгер
	zapLogger, err := initializeLogger()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer zapLogger.Sync()

	// Логируем информацию о версии
	version.LogVersionInfo(zapLogger, "GophKeeper client")

	// Создаем клиент
	gkClient, err := createClient(cfg, zapLogger)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer gkClient.Close()

	// Запускаем TUI приложение
	if err := runTUI(ctx, gkClient, zapLogger); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// loadConfiguration загружает конфигурацию клиента
func loadConfiguration() (*config.ClientConfig, error) {
	return config.LoadClientConfig()
}

// initializeLogger инициализирует логгер
func initializeLogger() (*zap.Logger, error) {
	return logger.NewDevelopmentLogger()
}

// createClient создает клиент GophKeeper
func createClient(cfg *config.ClientConfig, logger *zap.Logger) (*client.Client, error) {
	return client.NewClient(cfg, logger)
}

// runTUI запускает терминальный интерфейс пользователя.
func runTUI(ctx context.Context, gkClient *client.Client, logger *zap.Logger) error {
	model := client.NewTUIModel(gkClient, logger)

	// Создаем программу Bubble Tea
	program := tea.NewProgram(model, tea.WithAltScreen())

	// Запускаем программу
	_, err := program.Run()
	return err
}

// showHelp показывает справку по использованию
func showHelp() {
	// Используем шаблон из отдельного пакета
	templates.RenderHelpWithFallback(templates.DefaultHelpData())
}
