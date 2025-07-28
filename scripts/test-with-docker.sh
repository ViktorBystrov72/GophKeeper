#!/bin/bash

set -e

echo "🚀 Запуск тестов с базой данных в Docker"

# Останавливаем существующие контейнеры
echo "📦 Остановка существующих контейнеров..."
docker compose down -v

# Запускаем базу данных
echo "🐘 Запуск PostgreSQL..."
docker compose up -d postgres

# Ждем готовности базы данных
echo "⏳ Ожидание готовности базы данных..."
until docker compose exec -T postgres pg_isready -U gophkeeper; do
    echo "База данных еще не готова, ждем..."
    sleep 2
done

echo "✅ База данных готова!"

# Получаем IP адрес контейнера с базой данных
DB_HOST=$(docker compose exec -T postgres hostname -i | tr -d '\r')
echo "📍 IP адрес базы данных: $DB_HOST"

# Устанавливаем переменную окружения для тестов
export DATABASE_URI="postgres://gophkeeper:password@localhost:5432/gophkeeper?sslmode=disable"
echo "🔗 DATABASE_URI: $DATABASE_URI"

# Функция для запуска тестов storage
run_storage_tests() {
	echo "🧪 Запуск тестов storage локально..."
	go test -v -timeout=30s ./internal/storage/... -tags=integration
}

# Функция для запуска тестов gRPC
run_grpc_tests() {
	echo "🔗 Запуск тестов gRPC локально..."
	go test -v -timeout=30s ./internal/grpc/... -tags=integration
}

# Проверяем аргументы командной строки
if [ "$1" = "storage" ]; then
	run_storage_tests
elif [ "$1" = "grpc" ]; then
	run_grpc_tests
else
	# Запускаем все тесты по умолчанию
	run_storage_tests
	run_grpc_tests
fi

echo "✅ Тесты завершены!"

# Останавливаем контейнеры
echo "🛑 Остановка контейнеров..."
docker compose down 