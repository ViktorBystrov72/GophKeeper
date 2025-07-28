package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHashAndCheckPassword(t *testing.T) {
	s := NewService("test-secret")
	password := "supersecret"
	hash, err := s.HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	ok := s.CheckPassword(password, hash)
	require.True(t, ok)
	ok = s.CheckPassword("wrong", hash)
	require.False(t, ok)
}

func TestGenerateAndValidateToken(t *testing.T) {
	s := NewService("test-secret")
	userID := uuid.New()
	username := "testuser"
	token, expiresAt, err := s.GenerateToken(userID, username)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.True(t, expiresAt.After(time.Now()))

	claims, err := s.ValidateToken(token)
	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, username, claims.Username)
}

func TestRefreshToken(t *testing.T) {
	s := NewService("test-secret")
	userID := uuid.New()
	username := "testuser"
	token, _, err := s.GenerateToken(userID, username)
	require.NoError(t, err)

	newToken, newExp, err := s.RefreshToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, newToken)
	require.True(t, newExp.After(time.Now()))

	claims, err := s.ValidateToken(newToken)
	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
}
