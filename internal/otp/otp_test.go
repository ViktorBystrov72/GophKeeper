package otp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateSecret(t *testing.T) {
	s := NewService()
	secret, err := s.GenerateSecret()
	require.NoError(t, err)
	require.NotEmpty(t, secret)
}

func TestGenerateAndValidateCode(t *testing.T) {
	s := NewService()
	secret, err := s.GenerateSecret()
	require.NoError(t, err)

	code, err := s.GenerateCode(secret)
	require.NoError(t, err)
	require.Len(t, code, 6)

	// Валидный код
	ok, err := s.ValidateCode(secret, code)
	require.NoError(t, err)
	require.True(t, ok)

	// Невалидный код
	ok, err = s.ValidateCode(secret, "000000")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestGenerateQRCodeURL(t *testing.T) {
	s := NewService()
	secret, _ := s.GenerateSecret()
	url, err := s.GenerateQRCodeURL(secret, "GophKeeper", "testuser")
	require.NoError(t, err)
	require.Contains(t, url, "otpauth://totp/")
}

func TestGenerateBackupCodes(t *testing.T) {
	s := NewService()
	codes, err := s.GenerateBackupCodes(5)
	require.NoError(t, err)
	require.Len(t, codes, 5)
	for _, code := range codes {
		parts := strings.Split(code, "-")
		require.Len(t, parts, 2)
		require.Len(t, parts[0], 5)
		require.Len(t, parts[1], 5)
	}
}

func TestGetTimeRemaining(t *testing.T) {
	s := NewService()
	rem := s.GetTimeRemaining()
	require.GreaterOrEqual(t, rem, 0)
	require.LessOrEqual(t, rem, 30)
}
