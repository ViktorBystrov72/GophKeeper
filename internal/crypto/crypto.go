// Package crypto предоставляет функции шифрования и дешифрования данных.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
)

const (
	// AESKeySize размер ключа AES-256
	AESKeySize = 32
	// MaxRSABlockSize максимальный размер блока для RSA шифрования
	MaxRSABlockSize = 190 // для RSA-2048 с OAEP padding
)

// Service предоставляет методы для шифрования и дешифрования данных.
type Service struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewService создает новый сервис шифрования.
func NewService(privateKeyPEM, publicKeyPEM []byte) (*Service, error) {
	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	publicKey, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &Service{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// GenerateAESKey генерирует случайный ключ AES-256.
func GenerateAESKey() ([]byte, error) {
	key := make([]byte, AESKeySize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}
	return key, nil
}

// EncryptAES шифрует данные с помощью AES-GCM.
func EncryptAES(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// DecryptAES дешифрует данные с помощью AES-GCM.
func DecryptAES(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptRSA шифрует данные с помощью RSA-OAEP.
func (s *Service) EncryptRSA(data []byte) ([]byte, error) {
	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, s.publicKey, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt with RSA: %w", err)
	}
	return ciphertext, nil
}

// DecryptRSA дешифрует данные с помощью RSA-OAEP.
func (s *Service) DecryptRSA(ciphertext []byte) ([]byte, error) {
	hash := sha256.New()
	plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, s.privateKey, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt with RSA: %w", err)
	}
	return plaintext, nil
}

// EncryptLargeData шифрует данные произвольного размера, разбивая их на блоки.
// Для небольших данных использует RSA, для больших - комбинацию RSA + AES.
func (s *Service) EncryptLargeData(data []byte) ([]byte, error) {
	if len(data) <= MaxRSABlockSize {
		return s.EncryptRSA(data)
	}

	// Генерируем AES ключ
	aesKey, err := GenerateAESKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}

	// Шифруем данные с помощью AES
	encryptedData, err := EncryptAES(data, aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data with AES: %w", err)
	}

	// Шифруем AES ключ с помощью RSA
	encryptedKey, err := s.EncryptRSA(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt AES key: %w", err)
	}

	// Объединяем зашифрованный ключ и данные
	result := make([]byte, 4+len(encryptedKey)+len(encryptedData))

	// Первые 4 байта - размер зашифрованного ключа
	result[0] = byte(len(encryptedKey) >> 24)
	result[1] = byte(len(encryptedKey) >> 16)
	result[2] = byte(len(encryptedKey) >> 8)
	result[3] = byte(len(encryptedKey))

	copy(result[4:], encryptedKey)
	copy(result[4+len(encryptedKey):], encryptedData)

	return result, nil
}

// DecryptLargeData дешифрует данные, зашифрованные с помощью EncryptLargeData.
func (s *Service) DecryptLargeData(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 4 {
		// Попытка дешифровать как простой RSA блок
		return s.DecryptRSA(ciphertext)
	}

	// Читаем размер зашифрованного ключа
	keySize := int(ciphertext[0])<<24 | int(ciphertext[1])<<16 | int(ciphertext[2])<<8 | int(ciphertext[3])

	if len(ciphertext) < 4+keySize {
		// Попытка дешифровать как простой RSA блок
		return s.DecryptRSA(ciphertext)
	}

	// Извлекаем зашифрованный ключ и данные
	encryptedKey := ciphertext[4 : 4+keySize]
	encryptedData := ciphertext[4+keySize:]

	// Дешифруем AES ключ
	aesKey, err := s.DecryptRSA(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt AES key: %w", err)
	}

	// Дешифруем данные
	data, err := DecryptAES(encryptedData, aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return data, nil
}

// parsePrivateKey парсит приватный ключ из PEM формата.
func parsePrivateKey(keyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Попробуем PKCS8 формат
		keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		rsaKey, ok := keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}
		return rsaKey, nil
	}

	return key, nil
}

// parsePublicKey парсит публичный ключ из PEM формата.
func parsePublicKey(keyPEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	keyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaKey, ok := keyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaKey, nil
}
