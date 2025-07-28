package templates

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDefaultHelpData(t *testing.T) {
	data := DefaultHelpData()

	// Проверяем, что все поля заполнены
	if data.DefaultServer == "" {
		t.Error("DefaultServer should not be empty")
	}
	if data.DefaultGRPC == "" {
		t.Error("DefaultGRPC should not be empty")
	}
	if data.DefaultConfig == "" {
		t.Error("DefaultConfig should not be empty")
	}
	if data.DefaultLogLevel == "" {
		t.Error("DefaultLogLevel should not be empty")
	}

	// Проверяем конкретные значения
	expectedServer := "localhost:8080"
	if data.DefaultServer != expectedServer {
		t.Errorf("DefaultServer expected %s, got %s", expectedServer, data.DefaultServer)
	}

	expectedGRPC := "localhost:8081"
	if data.DefaultGRPC != expectedGRPC {
		t.Errorf("DefaultGRPC expected %s, got %s", expectedGRPC, data.DefaultGRPC)
	}

	expectedConfig := "./config.json"
	if data.DefaultConfig != expectedConfig {
		t.Errorf("DefaultConfig expected %s, got %s", expectedConfig, data.DefaultConfig)
	}

	expectedLogLevel := "info"
	if data.DefaultLogLevel != expectedLogLevel {
		t.Errorf("DefaultLogLevel expected %s, got %s", expectedLogLevel, data.DefaultLogLevel)
	}
}

func TestRenderHelp(t *testing.T) {
	data := HelpData{
		DefaultServer:   "test-server:8080",
		DefaultGRPC:     "test-grpc:8081",
		DefaultConfig:   "test-config.json",
		DefaultLogLevel: "debug",
	}

	// Создаем буфер для захвата вывода
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Выполняем рендеринг
	err := RenderHelp(data)

	w.Close()
	os.Stdout = oldStdout

	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("RenderHelp returned error: %v", err)
	}

	// Проверяем, что вывод содержит ожидаемые значения
	if !strings.Contains(output, "test-server:8080") {
		t.Error("Output should contain custom server address")
	}
	if !strings.Contains(output, "test-grpc:8081") {
		t.Error("Output should contain custom gRPC address")
	}
	if !strings.Contains(output, "test-config.json") {
		t.Error("Output should contain custom config path")
	}
	if !strings.Contains(output, "debug") {
		t.Error("Output should contain custom log level")
	}

	// Проверяем, что вывод содержит основные элементы справки
	if !strings.Contains(output, "GophKeeper - Secure Password Manager") {
		t.Error("Output should contain application name")
	}
	if !strings.Contains(output, "Usage:") {
		t.Error("Output should contain usage section")
	}
	if !strings.Contains(output, "Options:") {
		t.Error("Output should contain options section")
	}
}

func TestRenderHelpWithFallback(t *testing.T) {
	data := HelpData{
		DefaultServer:   "test-server:8080",
		DefaultGRPC:     "test-grpc:8081",
		DefaultConfig:   "test-config.json",
		DefaultLogLevel: "debug",
	}

	// Создаем буфер для захвата вывода
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Выполняем рендеринг с fallback
	RenderHelpWithFallback(data)

	// Восстанавливаем stdout
	w.Close()
	os.Stdout = oldStdout

	buf.ReadFrom(r)
	output := buf.String()

	// Проверяем, что вывод содержит ожидаемые значения
	if !strings.Contains(output, "test-server:8080") {
		t.Error("Output should contain custom server address")
	}
	if !strings.Contains(output, "test-grpc:8081") {
		t.Error("Output should contain custom gRPC address")
	}
	if !strings.Contains(output, "test-config.json") {
		t.Error("Output should contain custom config path")
	}
	if !strings.Contains(output, "debug") {
		t.Error("Output should contain custom log level")
	}
}

func TestHelpTemplateContainsRequiredFields(t *testing.T) {
	// Проверяем, что шаблон содержит все необходимые поля
	requiredFields := []string{
		"{{.DefaultServer}}",
		"{{.DefaultGRPC}}",
		"{{.DefaultConfig}}",
		"{{.DefaultLogLevel}}",
	}

	for _, field := range requiredFields {
		if !strings.Contains(HelpTemplate, field) {
			t.Errorf("HelpTemplate should contain field %s", field)
		}
	}
}
