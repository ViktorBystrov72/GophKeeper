-- +goose Up
-- +goose StatementBegin

-- Создание таблицы пользователей
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Создание индекса на username для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Создание таблицы для хранения данных
CREATE TABLE IF NOT EXISTS data_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('credentials', 'text', 'binary', 'card')),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    encrypted_data BYTEA NOT NULL,
    metadata TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version BIGINT DEFAULT 1,
    
    CONSTRAINT unique_user_name UNIQUE(user_id, name)
);

-- Создание индексов для оптимизации запросов
CREATE INDEX IF NOT EXISTS idx_data_entries_user_id ON data_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_data_entries_type ON data_entries(type);
CREATE INDEX IF NOT EXISTS idx_data_entries_updated_at ON data_entries(updated_at);

-- Создание таблицы для отслеживания удаленных записей (для синхронизации)
CREATE TABLE IF NOT EXISTS deleted_entries (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Создание индекса на user_id и deleted_at для синхронизации
CREATE INDEX IF NOT EXISTS idx_deleted_entries_user_deleted ON deleted_entries(user_id, deleted_at);

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггеры для автоматического обновления updated_at
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_data_entries_updated_at 
    BEFORE UPDATE ON data_entries 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Удаление триггеров
DROP TRIGGER IF EXISTS update_data_entries_updated_at ON data_entries;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Удаление функции
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Удаление таблиц
DROP TABLE IF EXISTS deleted_entries;
DROP TABLE IF EXISTS data_entries;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd 