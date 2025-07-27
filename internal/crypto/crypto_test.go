package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
)

func generateTestKeys(t *testing.T) ([]byte, []byte) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})

	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	return privPEM, pubPEM
}

func TestAES(t *testing.T) {
	key, err := GenerateAESKey()
	require.NoError(t, err)
	data := []byte("hello world")
	enc, err := EncryptAES(data, key)
	require.NoError(t, err)
	dec, err := DecryptAES(enc, key)
	require.NoError(t, err)
	require.Equal(t, data, dec)
}

func TestRSA(t *testing.T) {
	priv, pub := generateTestKeys(t)
	s, err := NewService(priv, pub)
	require.NoError(t, err)
	data := []byte("test data")
	enc, err := s.EncryptRSA(data)
	require.NoError(t, err)
	dec, err := s.DecryptRSA(enc)
	require.NoError(t, err)
	require.Equal(t, data, dec)
}

func TestEncryptLargeData(t *testing.T) {
	priv, pub := generateTestKeys(t)
	s, err := NewService(priv, pub)
	require.NoError(t, err)
	data := make([]byte, 512)
	_, err = rand.Read(data)
	require.NoError(t, err)
	enc, err := s.EncryptLargeData(data)
	require.NoError(t, err)
	dec, err := s.DecryptLargeData(enc)
	require.NoError(t, err)
	require.Equal(t, data, dec)
}
