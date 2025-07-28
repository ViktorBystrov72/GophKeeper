// Package templates содержит шаблоны для различных частей приложения
package templates

import (
	"fmt"
	"os"
	"text/template"
)

// HelpTemplate определяет шаблон для справки клиента
const HelpTemplate = `GophKeeper - Secure Password Manager

Usage:
  gophkeeper-client [options]

Options:
  -server string     Server address (default "{{.DefaultServer}}")
  -grpc string       gRPC server address (default "{{.DefaultGRPC}}")
  -tls               Enable TLS connection
  -cert string       TLS certificate file
  -config string     Configuration file path (default "{{.DefaultConfig}}")
  -log string        Log level (default "{{.DefaultLogLevel}}")
  -v, --version      Show version information
  -h, --help         Show this help message

Commands:
  version            Show version information
  help               Show this help message

Interactive Mode:
  Run without arguments to start the interactive TUI interface
`

// HelpData содержит данные для шаблона справки
type HelpData struct {
	DefaultServer   string
	DefaultGRPC     string
	DefaultConfig   string
	DefaultLogLevel string
}

// DefaultHelpData возвращает данные по умолчанию для справки
func DefaultHelpData() HelpData {
	return HelpData{
		DefaultServer:   "localhost:8080",
		DefaultGRPC:     "localhost:8081",
		DefaultConfig:   "./config.json",
		DefaultLogLevel: "info",
	}
}

// RenderHelp отображает справку с использованием шаблона
func RenderHelp(data HelpData) error {
	// Парсим шаблон
	tmpl, err := template.New("help").Parse(HelpTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse help template: %w", err)
	}

	// Выполняем шаблон
	if err := tmpl.Execute(os.Stdout, data); err != nil {
		return fmt.Errorf("failed to execute help template: %w", err)
	}

	return nil
}

// RenderHelpWithFallback отображает справку с fallback при ошибках
func RenderHelpWithFallback(data HelpData) {
	if err := RenderHelp(data); err != nil {
		// Fallback при ошибке
		fmt.Printf("GophKeeper - Secure Password Manager\n")
		fmt.Printf("Usage: gophkeeper-client [options]\n")
		fmt.Printf("Use 'gophkeeper-client help' for more information\n")
	}
} 