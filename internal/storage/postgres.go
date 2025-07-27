// Package storage предоставляет интерфейсы и реализации для хранения данных.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/GophKeeper/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Storage определяет интерфейс для хранения данных.
type Storage interface {
	// Методы для работы с пользователями
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error)

	// Методы для работы с данными
	CreateDataEntry(ctx context.Context, entry *models.DataEntry) error
	GetDataEntry(ctx context.Context, userID, entryID uuid.UUID) (*models.DataEntry, error)
	GetDataEntries(ctx context.Context, userID uuid.UUID, dataType *models.DataType) ([]models.DataEntry, error)
	UpdateDataEntry(ctx context.Context, entry *models.DataEntry) error
	DeleteDataEntry(ctx context.Context, userID, entryID uuid.UUID) error

	// Методы для синхронизации
	GetDataEntriesAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]models.DataEntry, error)
	GetDeletedEntriesAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]uuid.UUID, error)

	// Закрытие соединения
	Close()
}

// PostgresStorage реализует интерфейс Storage для PostgreSQL.
type PostgresStorage struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresStorage создает новое подключение к PostgreSQL.
func NewPostgresStorage(ctx context.Context, databaseURI string, logger *zap.Logger) (*PostgresStorage, error) {
	// Выполняем миграции перед созданием пула соединений
	if err := RunMigrations(ctx, databaseURI, "../../migrations"); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

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

// RunMigrations выполняет миграции базы данных
func RunMigrations(ctx context.Context, dsn string, migrationsDir string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Получаем абсолютный путь к директории миграций
	absPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for migrations: %w", err)
	}

	if err := goose.Up(db, absPath); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// CreateUser создает нового пользователя.
func (s *PostgresStorage) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`

	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := s.pool.Exec(ctx, query, user.ID, user.Username, user.PasswordHash, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		// Проверяем на дубликат имени пользователя
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("username already exists: %w", err)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
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

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
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

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// CreateDataEntry создает новую запись данных.
func (s *PostgresStorage) CreateDataEntry(ctx context.Context, entry *models.DataEntry) error {
	query := `
		INSERT INTO data_entries (id, user_id, type, name, description, encrypted_data, metadata, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	entry.ID = uuid.New()
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()
	entry.Version = 1

	_, err := s.pool.Exec(ctx, query,
		entry.ID, entry.UserID, entry.Type, entry.Name,
		entry.Description, entry.EncryptedData, entry.Metadata,
		entry.CreatedAt, entry.UpdatedAt, entry.Version,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("entry with this name already exists: %w", err)
		}
		return fmt.Errorf("failed to create data entry: %w", err)
	}

	return nil
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

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("data entry not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get data entry: %w", err)
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
	if err != nil {
		return nil, fmt.Errorf("failed to query data entries: %w", err)
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
		if err != nil {
			return nil, fmt.Errorf("failed to scan data entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
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

	if err != nil {
		return fmt.Errorf("failed to update data entry: %w", err)
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
	if err != nil {
		return fmt.Errorf("failed to delete data entry: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("data entry not found")
	}

	// Добавляем запись в таблицу удаленных для синхронизации
	insertQuery := `INSERT INTO deleted_entries (id, user_id, deleted_at) VALUES ($1, $2, NOW())`
	_, err = tx.Exec(ctx, insertQuery, entryID, userID)
	if err != nil {
		return fmt.Errorf("failed to insert deleted entry: %w", err)
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
	if err != nil {
		return nil, fmt.Errorf("failed to query data entries: %w", err)
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
		if err != nil {
			return nil, fmt.Errorf("failed to scan data entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetDeletedEntriesAfter получает ID записей, удаленных после указанного времени.
func (s *PostgresStorage) GetDeletedEntriesAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]uuid.UUID, error) {
	query := `
		SELECT id 
		FROM deleted_entries 
		WHERE user_id = $1 AND deleted_at > $2
		ORDER BY deleted_at ASC`

	rows, err := s.pool.Query(ctx, query, userID, after)
	if err != nil {
		return nil, fmt.Errorf("failed to query deleted entries: %w", err)
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

	return deletedIDs, rows.Err()
}

// Close закрывает соединение с базой данных.
func (s *PostgresStorage) Close() {
	s.pool.Close()
}
