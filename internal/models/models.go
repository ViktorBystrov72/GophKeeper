// Package models определяет структуры данных для GophKeeper.
package models

import (
	"time"

	"github.com/google/uuid"
)

// DataType представляет тип хранимых данных.
type DataType string

const (
	DataTypeCredentials DataType = "credentials" // пары логин/пароль
	DataTypeText        DataType = "text"        // произвольные текстовые данные
	DataTypeBinary      DataType = "binary"      // произвольные бинарные данные
	DataTypeCard        DataType = "card"        // данные банковских карт
)

// User представляет пользователя в системе.
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username" validate:"required,min=3,max=50"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// DataEntry представляет запись сохраненных данных.
type DataEntry struct {
	ID            uuid.UUID `json:"id" db:"id"`
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	Type          DataType  `json:"type" db:"type"`
	Name          string    `json:"name" db:"name" validate:"required,min=1,max=100"`
	Description   string    `json:"description" db:"description"`
	EncryptedData []byte    `json:"encrypted_data" db:"encrypted_data"`
	Metadata      string    `json:"metadata" db:"metadata"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	Version       int64     `json:"version" db:"version"`
}

// Credentials представляет пары логин/пароль.
type Credentials struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
	URL      string `json:"url,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

// TextData представляет произвольные текстовые данные.
type TextData struct {
	Content string `json:"content" validate:"required"`
	Notes   string `json:"notes,omitempty"`
}

// BinaryData представляет произвольные бинарные данные.
type BinaryData struct {
	Filename string `json:"filename" validate:"required"`
	Content  []byte `json:"content" validate:"required"`
	MimeType string `json:"mime_type,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

// CardData представляет данные банковских карт.
type CardData struct {
	Number     string `json:"number" validate:"required,creditcard"`
	ExpiryDate string `json:"expiry_date" validate:"required"`
	Holder     string `json:"holder" validate:"required"`
	CVV        string `json:"cvv" validate:"required,len=3"`
	PIN        string `json:"pin,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// AuthRequest представляет запрос на аутентификацию.
type AuthRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

// AuthResponse представляет ответ на аутентификацию.
type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

// RegisterRequest представляет запрос на регистрацию пользователя.
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

// CreateDataRequest представляет запрос на создание данных.
type CreateDataRequest struct {
	Type        DataType    `json:"type" validate:"required,oneof=credentials text binary card"`
	Name        string      `json:"name" validate:"required,min=1,max=100"`
	Description string      `json:"description"`
	Data        interface{} `json:"data" validate:"required"`
	Metadata    string      `json:"metadata"`
}

// UpdateDataRequest представляет запрос на обновление данных.
type UpdateDataRequest struct {
	ID          uuid.UUID   `json:"id" validate:"required"`
	Name        string      `json:"name" validate:"required,min=1,max=100"`
	Description string      `json:"description"`
	Data        interface{} `json:"data" validate:"required"`
	Metadata    string      `json:"metadata"`
	Version     int64       `json:"version" validate:"required"`
}

// DataResponse представляет ответ с данными.
type DataResponse struct {
	ID          uuid.UUID   `json:"id"`
	Type        DataType    `json:"type"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Data        interface{} `json:"data"`
	Metadata    string      `json:"metadata"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Version     int64       `json:"version"`
}

// SyncRequest представляет запрос на синхронизацию.
type SyncRequest struct {
	LastSyncTime time.Time `json:"last_sync_time"`
}

// SyncResponse представляет ответ на синхронизацию.
type SyncResponse struct {
	Data         []DataResponse `json:"data"`
	DeletedIDs   []uuid.UUID    `json:"deleted_ids"`
	LastSyncTime time.Time      `json:"last_sync_time"`
}

// OTPRequest представляет запрос на генерацию OTP.
type OTPRequest struct {
	Secret string `json:"secret" validate:"required"`
}

// OTPResponse представляет ответ с OTP кодом.
type OTPResponse struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ErrorResponse представляет ответ с ошибкой.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}
