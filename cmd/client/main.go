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
	"github.com/GophKeeper/internal/version"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	// Проверяем флаги команды ПЕРЕД инициализации логгера
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			// Устанавливаем значения по умолчанию для версии
			version.SetDefaults()
			version.PrintVersionInfo("GophKeeper Client")
			return
		case "help", "--help", "-h":
			showHelp()
			return
		}
	}

	cfg, err := config.LoadClientConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	zapLogger, err := logger.NewDevelopmentLogger()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer zapLogger.Sync()

	version.LogVersionInfo(zapLogger, "GophKeeper client")

	gkClient, err := client.NewClient(cfg, zapLogger)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer gkClient.Close()

	// Запускаем TUI приложение
	if err := runTUI(ctx, gkClient, zapLogger); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
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

// showHelp показывает справку по использованию.
func showHelp() {
	fmt.Printf("GophKeeper - Secure Password Manager\n\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("  gophkeeper-client [options]\n\n")
	fmt.Printf("Options:\n")
	fmt.Printf("  -server string     Server address (default \"localhost:8080\")\n")
	fmt.Printf("  -grpc string       gRPC server address (default \"localhost:8081\")\n")
	fmt.Printf("  -tls               Enable TLS connection\n")
	fmt.Printf("  -cert string       TLS certificate file\n")
	fmt.Printf("  -config string     Configuration file path (default \"./config.json\")\n")
	fmt.Printf("  -log string        Log level (default \"info\")\n")
	fmt.Printf("  -v, --version      Show version information\n")
	fmt.Printf("  -h, --help         Show this help message\n\n")
	fmt.Printf("Commands:\n")
	fmt.Printf("  version            Show version information\n")
	fmt.Printf("  help               Show this help message\n\n")
	fmt.Printf("Interactive Mode:\n")
	fmt.Printf("  Run without arguments to start the interactive TUI interface\n")
}
