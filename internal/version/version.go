// Package version содержит логику для управления версионированием приложения.
package version

import (
	"fmt"

	"go.uber.org/zap"
)

// Переменные для версионирования
var (
	BuildVersion string
	BuildDate    string
	BuildCommit  string
)

// SetDefaults устанавливает значения по умолчанию для переменных версии
func SetDefaults() {
	if BuildVersion == "" {
		BuildVersion = "N/A"
	}
	if BuildDate == "" {
		BuildDate = "N/A"
	}
	if BuildCommit == "" {
		BuildCommit = "N/A"
	}
}

// LogVersionInfo логирует информацию о версии через zap логгер
func LogVersionInfo(logger *zap.Logger, appName string) {
	SetDefaults()
	logger.Info("Starting "+appName,
		zap.String("version", BuildVersion),
		zap.String("build_date", BuildDate),
		zap.String("build_commit", BuildCommit),
	)
}

// PrintVersionInfo выводит информацию о версии в стандартный вывод
func PrintVersionInfo(appName string) {
	SetDefaults()
	fmt.Printf("%s\n", appName)
	fmt.Printf("Version: %s\n", BuildVersion)
	fmt.Printf("Build Date: %s\n", BuildDate)
	fmt.Printf("Build Commit: %s\n", BuildCommit)
}
