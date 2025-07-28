#!/bin/bash

# Скрипт для запуска сервера GophKeeper

echo "🚀 Запуск сервера GophKeeper..."

# Проверяем, что база данных запущена
if ! docker compose ps postgres | grep -q "Up"; then
    echo "📦 Запуск базы данных..."
    docker compose up -d postgres
    sleep 5
fi

# Запускаем сервер с правильными переменными окружения
echo "🔧 Запуск сервера..."
DATABASE_URI="postgres://gophkeeper:password@localhost:5432/gophkeeper?sslmode=disable" \
JWT_SECRET="your-secret-key" \
ENCRYPTION_KEY="your-encryption-key" \
LOG_LEVEL="info" \
SERVER_ADDRESS=":8080" \
GRPC_ADDRESS=":9090" \
./bin/server 