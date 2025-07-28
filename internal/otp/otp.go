// Package otp предоставляет функциональность для генерации одноразовых паролей.
package otp

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// Service предоставляет методы для работы с OTP.
type Service struct{}

// NewService создает новый сервис OTP.
func NewService() *Service {
	return &Service{}
}

// GenerateSecret генерирует секретный ключ для OTP.
func (s *Service) GenerateSecret() (string, error) {
	secret := make([]byte, 20) // 160 бит
	_, err := rand.Read(secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate random secret: %w", err)
	}

	return base32.StdEncoding.EncodeToString(secret), nil
}

// GenerateCode генерирует текущий TOTP код из секрета.
func (s *Service) GenerateCode(secret string) (string, error) {
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP code: %w", err)
	}

	return code, nil
}

// ValidateCode проверяет TOTP код.
func (s *Service) ValidateCode(secret, code string) (bool, error) {
	valid := totp.Validate(code, secret)
	return valid, nil
}

// GenerateQRCodeURL генерирует URL для QR кода для настройки приложения аутентификатора.
func (s *Service) GenerateQRCodeURL(secret, issuer, accountName string) (string, error) {
	key, err := otp.NewKeyFromURL(fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		url.QueryEscape(issuer),
		url.QueryEscape(accountName),
		secret,
		url.QueryEscape(issuer)))
	if err != nil {
		return "", fmt.Errorf("failed to create OTP key: %w", err)
	}

	return key.URL(), nil
}

// GetTimeRemaining возвращает количество секунд до истечения текущего кода.
func (s *Service) GetTimeRemaining() int {
	period := 30 // TOTP период в секундах
	return period - int(time.Now().Unix()%int64(period))
}

// GenerateBackupCodes генерирует резервные коды для восстановления доступа.
func (s *Service) GenerateBackupCodes(count int) ([]string, error) {
	if count <= 0 {
		count = 10 // по умолчанию 10 кодов
	}

	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := s.generateBackupCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code %d: %w", i, err)
		}
		codes[i] = code
	}

	return codes, nil
}

// generateBackupCode генерирует один резервный код.
func (s *Service) generateBackupCode() (string, error) {
	bytes := make([]byte, 5) // 10 символов в base32
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Кодируем в base32 и убираем padding
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)

	// Гарантируем длину 10 символов
	if len(code) < 10 {
		code = code + strings.Repeat("0", 10-len(code))
	}

	return fmt.Sprintf("%s-%s", code[:5], code[5:10]), nil
}
