// Package storage предоставляет интерфейсы и реализации для хранения данных.
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GophKeeper/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// UserRepository определяет интерфейс для работы с пользователями
type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

// DataRepository определяет интерфейс для работы с данными
type DataRepository interface {
	CreateDataEntry(ctx context.Context, entry *models.DataEntry) error
	GetDataEntry(ctx context.Context, userID, entryID uuid.UUID) (*models.DataEntry, error)
	GetDataEntries(ctx context.Context, userID uuid.UUID, dataType *models.DataType) ([]models.DataEntry, error)
	UpdateDataEntry(ctx context.Context, entry *models.DataEntry) error
	DeleteDataEntry(ctx context.Context, userID, entryID uuid.UUID) error
}

// SyncRepository определяет интерфейс для синхронизации данных
type SyncRepository interface {
	GetDataEntriesAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]models.DataEntry, error)
	GetDeletedEntriesAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]uuid.UUID, error)
}

// ConnectionManager определяет интерфейс для управления соединением
type ConnectionManager interface {
	Close()
}

// Storage объединяет все репозитории в один интерфейс
type Storage interface {
	UserRepository
	DataRepository
	SyncRepository
	ConnectionManager
}

// PostgresStorage реализует все интерфейсы для PostgreSQL.
type PostgresStorage struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresStorage создает новое подключение к PostgreSQL.
func NewPostgresStorage(ctx context.Context, databaseURI string, logger *zap.Logger) (*PostgresStorage, error) {
	pool, err := pgxpool.New(ctx, databaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем соединение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStorage{
		pool:   pool,
		logger: logger,
	}, nil
}

// NewPostgresStorageForTests создает новое подключение к PostgreSQL для тестов без выполнения миграций
func NewPostgresStorageForTests(ctx context.Context, databaseURI string, logger *zap.Logger) (*PostgresStorage, error) {
	pool, err := pgxpool.New(ctx, databaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем соединение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStorage{
		pool:   pool,
		logger: logger,
	}, nil
}

// prepareNewEntity подготавливает ID и временные метки для новой сущности
func (s *PostgresStorage) prepareNewEntity() (uuid.UUID, time.Time, time.Time) {
	id := uuid.New()
	now := time.Now()
	return id, now, now
}

// prepareNewDataEntry подготавливает ID, временные метки и версию для новой записи данных
func (s *PostgresStorage) prepareNewDataEntry() (uuid.UUID, time.Time, time.Time, int64) {
	id := uuid.New()
	now := time.Now()
	return id, now, now, 1
}

// handleDBError обрабатывает ошибки базы данных с проверкой на уникальность
func (s *PostgresStorage) handleDBError(err error, uniqueViolationMsg string) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		return fmt.Errorf("%s: %w", uniqueViolationMsg, err)
	}
	return fmt.Errorf("database operation failed: %w", err)
}

// handleQueryRowError обрабатывает ошибки QueryRow с проверкой на отсутствие строк
func (s *PostgresStorage) handleQueryRowError(err error, notFoundMsg string, operationMsg string) error {
	if err == nil {
		return nil
	}

	if err == pgx.ErrNoRows {
		return fmt.Errorf("%s: %w", notFoundMsg, err)
	}
	return fmt.Errorf("%s: %w", operationMsg, err)
}

// handleQueryError обрабатывает ошибки Query с дополнительной информацией
func (s *PostgresStorage) handleQueryError(err error, operationMsg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operationMsg, err)
}

// handleScanError обрабатывает ошибки сканирования результатов
func (s *PostgresStorage) handleScanError(err error, operationMsg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operationMsg, err)
}

// handleRowsError обрабатывает ошибки итерации по строкам
func (s *PostgresStorage) handleRowsError(err error, operationMsg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operationMsg, err)
}

// handleExecError обрабатывает ошибки Exec с проверкой на уникальность
func (s *PostgresStorage) handleExecError(err error, uniqueViolationMsg string, operationMsg string) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		return fmt.Errorf("%s: %w", uniqueViolationMsg, err)
	}
	return fmt.Errorf("%s: %w", operationMsg, err)
}

// CreateUser создает нового пользователя.
func (s *PostgresStorage) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`

	user.ID, user.CreatedAt, user.UpdatedAt = s.prepareNewEntity()

	_, err := s.pool.Exec(ctx, query, user.ID, user.Username, user.PasswordHash, user.CreatedAt, user.UpdatedAt)
	return s.handleExecError(err, "username already exists", "failed to create user")
}

// GetUserByUsername получает пользователя по имени.
func (s *PostgresStorage) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users 
		WHERE username = $1`

	var user models.User
	err := s.pool.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err := s.handleQueryRowError(err, "user not found", "failed to get user"); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByID получает пользователя по ID.
func (s *PostgresStorage) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users 
		WHERE id = $1`

	var user models.User
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Username, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err := s.handleQueryRowError(err, "user not found", "failed to get user"); err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateDataEntry создает новую запись данных.
func (s *PostgresStorage) CreateDataEntry(ctx context.Context, entry *models.DataEntry) error {
	query := `
		INSERT INTO data_entries (id, user_id, type, name, description, encrypted_data, metadata, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	entry.ID, entry.CreatedAt, entry.UpdatedAt, entry.Version = s.prepareNewDataEntry()

	_, err := s.pool.Exec(ctx, query,
		entry.ID, entry.UserID, entry.Type, entry.Name,
		entry.Description, entry.EncryptedData, entry.Metadata,
		entry.CreatedAt, entry.UpdatedAt, entry.Version,
	)

	return s.handleExecError(err, "entry with this name already exists", "failed to create data entry")
}

// GetDataEntry получает запись данных по ID.
func (s *PostgresStorage) GetDataEntry(ctx context.Context, userID, entryID uuid.UUID) (*models.DataEntry, error) {
	query := `
		SELECT id, user_id, type, name, description, encrypted_data, metadata, created_at, updated_at, version
		FROM data_entries 
		WHERE id = $1 AND user_id = $2`

	var entry models.DataEntry
	err := s.pool.QueryRow(ctx, query, entryID, userID).Scan(
		&entry.ID, &entry.UserID, &entry.Type, &entry.Name,
		&entry.Description, &entry.EncryptedData, &entry.Metadata,
		&entry.CreatedAt, &entry.UpdatedAt, &entry.Version,
	)

	if err := s.handleQueryRowError(err, "data entry not found", "failed to get data entry"); err != nil {
		return nil, err
	}

	return &entry, nil
}

// GetDataEntries получает все записи данных пользователя.
func (s *PostgresStorage) GetDataEntries(ctx context.Context, userID uuid.UUID, dataType *models.DataType) ([]models.DataEntry, error) {
	var query string
	var args []interface{}

	if dataType != nil {
		query = `
			SELECT id, user_id, type, name, description, encrypted_data, metadata, created_at, updated_at, version
			FROM data_entries 
			WHERE user_id = $1 AND type = $2
			ORDER BY created_at DESC`
		args = []interface{}{userID, *dataType}
	} else {
		query = `
			SELECT id, user_id, type, name, description, encrypted_data, metadata, created_at, updated_at, version
			FROM data_entries 
			WHERE user_id = $1
			ORDER BY created_at DESC`
		args = []interface{}{userID}
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err := s.handleQueryError(err, "failed to query data entries"); err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.DataEntry
	for rows.Next() {
		var entry models.DataEntry
		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Type, &entry.Name,
			&entry.Description, &entry.EncryptedData, &entry.Metadata,
			&entry.CreatedAt, &entry.UpdatedAt, &entry.Version,
		)
		if err := s.handleScanError(err, "failed to scan data entry"); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := s.handleRowsError(rows.Err(), "error during rows iteration"); err != nil {
		return nil, err
	}

	return entries, nil
}

// UpdateDataEntry обновляет запись данных с проверкой версии.
func (s *PostgresStorage) UpdateDataEntry(ctx context.Context, entry *models.DataEntry) error {
	query := `
		UPDATE data_entries 
		SET name = $1, description = $2, encrypted_data = $3, metadata = $4, version = version + 1
		WHERE id = $5 AND user_id = $6 AND version = $7`

	result, err := s.pool.Exec(ctx, query,
		entry.Name, entry.Description, entry.EncryptedData, entry.Metadata,
		entry.ID, entry.UserID, entry.Version,
	)

	if err := s.handleExecError(err, "entry with this name already exists", "failed to update data entry"); err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("data entry not found or version mismatch")
	}

	// Обновляем версию в объекте
	entry.Version++
	entry.UpdatedAt = time.Now()

	return nil
}

// DeleteDataEntry удаляет запись данных.
func (s *PostgresStorage) DeleteDataEntry(ctx context.Context, userID, entryID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Удаляем запись
	deleteQuery := `DELETE FROM data_entries WHERE id = $1 AND user_id = $2`
	result, err := tx.Exec(ctx, deleteQuery, entryID, userID)
	if err := s.handleExecError(err, "", "failed to delete data entry"); err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("data entry not found")
	}

	// Добавляем запись в таблицу удаленных для синхронизации
	insertQuery := `INSERT INTO deleted_entries (id, user_id, deleted_at) VALUES ($1, $2, NOW())`
	_, err = tx.Exec(ctx, insertQuery, entryID, userID)
	if err := s.handleExecError(err, "", "failed to insert deleted entry"); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetDataEntriesAfter получает записи данных, измененные после указанного времени.
func (s *PostgresStorage) GetDataEntriesAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]models.DataEntry, error) {
	query := `
		SELECT id, user_id, type, name, description, encrypted_data, metadata, created_at, updated_at, version
		FROM data_entries 
		WHERE user_id = $1 AND updated_at > $2
		ORDER BY updated_at ASC`

	rows, err := s.pool.Query(ctx, query, userID, after)
	if err := s.handleQueryError(err, "failed to query data entries"); err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.DataEntry
	for rows.Next() {
		var entry models.DataEntry
		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Type, &entry.Name,
			&entry.Description, &entry.EncryptedData, &entry.Metadata,
			&entry.CreatedAt, &entry.UpdatedAt, &entry.Version,
		)
		if err := s.handleScanError(err, "failed to scan data entry"); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := s.handleRowsError(rows.Err(), "error during rows iteration"); err != nil {
		return nil, err
	}

	return entries, nil
}

// GetDeletedEntriesAfter получает ID записей, удаленных после указанного времени.
func (s *PostgresStorage) GetDeletedEntriesAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]uuid.UUID, error) {
	query := `
		SELECT id 
		FROM deleted_entries 
		WHERE user_id = $1 AND deleted_at > $2
		ORDER BY deleted_at ASC`

	rows, err := s.pool.Query(ctx, query, userID, after)
	if err := s.handleQueryError(err, "failed to query deleted entries"); err != nil {
		return nil, err
	}
	defer rows.Close()

	var deletedIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan deleted entry ID: %w", err)
		}
		deletedIDs = append(deletedIDs, id)
	}

	if err := s.handleRowsError(rows.Err(), "error during rows iteration"); err != nil {
		return nil, err
	}

	return deletedIDs, nil
}

// Close закрывает соединение с базой данных.
func (s *PostgresStorage) Close() {
	s.pool.Close()
}
