VERSION ?= N/A
BUILD_DATE ?= $(shell date '+%Y-%m-%d %H:%M:%S')
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo N/A)

LDFLAGS = -X github.com/GophKeeper/internal/version.BuildVersion=$(VERSION) -X 'github.com/GophKeeper/internal/version.BuildDate=$(BUILD_DATE)' -X github.com/GophKeeper/internal/version.BuildCommit=$(BUILD_COMMIT)

# Цели по умолчанию
.PHONY: help
help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: build-server build-client ## Собрать сервер и клиент

.PHONY: build-with-version
build-with-version: VERSION = v1.0.0
build-with-version: build ## Собрать с версией v1.0.0

.PHONY: build-server
build-server: ## Собрать сервер
	go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server

.PHONY: build-client
build-client: ## Собрать клиент
	go build -ldflags "$(LDFLAGS)" -o bin/client ./cmd/client

.PHONY: run-server
run-server: ## Запустить сервер
	go run -ldflags "$(LDFLAGS)" ./cmd/server

.PHONY: run-client
run-client: ## Запустить клиент
	go run -ldflags "$(LDFLAGS)" ./cmd/client

.PHONY: test
test: ## Запустить unit тесты
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-all
test-all: test test-integration ## Запустить все тесты (unit + интеграционные)

.PHONY: test-integration
test-integration: ## Запустить интеграционные тесты с Docker
	./scripts/test-with-docker.sh

.PHONY: test-unit
test-unit: ## Запустить только unit тесты
	go test -v -race -coverprofile=coverage.out ./... -short

.PHONY: test-storage
test-storage: ## Запустить тесты storage с реальной БД
	./scripts/test-with-docker.sh storage

.PHONY: test-grpc
test-grpc: ## Запустить тесты gRPC с реальной БД
	./scripts/test-with-docker.sh grpc

.PHONY: start-db
start-db: ## Запустить базу данных
	docker compose up -d postgres

.PHONY: start-server
start-server: ## Запустить сервер
	./scripts/start-server.sh

.PHONY: start-server-bg
start-server-bg: ## Запустить сервер в фоновом режиме
	@echo "🔧 Запуск сервера в фоновом режиме..."
	@DATABASE_URI="postgres://gophkeeper:password@localhost:5432/gophkeeper?sslmode=disable" \
	JWT_SECRET="your-secret-key" \
	ENCRYPTION_KEY="your-encryption-key" \
	LOG_LEVEL="info" \
	SERVER_ADDRESS=":8080" \
	GRPC_ADDRESS=":9090" \
	./bin/server > server.log 2>&1 &
	@echo "✅ Сервер запущен в фоновом режиме (логи в server.log)"

.PHONY: start-client
start-client: ## Запустить клиент
	./bin/client -grpc localhost:9090

.PHONY: stop-server
stop-server: ## Остановить сервер
	@echo "🛑 Остановка сервера..."
	@pkill -f "./bin/server" || echo "Сервер не был запущен"
	@echo "✅ Сервер остановлен"

.PHONY: stop-db
stop-db: ## Остановить базу данных
	@echo "🛑 Остановка базы данных..."
	@docker compose stop postgres
	@echo "✅ База данных остановлена"

.PHONY: stop-all
stop-all: stop-server stop-db ## Остановить всю систему
	@echo "🛑 Остановка всей системы GophKeeper..."
	@echo "✅ Система остановлена"

.PHONY: run
run: start-db start-server ## Запустить полную систему (БД + сервер)

.PHONY: ui
ui: start-db start-server-bg ## Запустить полную систему с UI (БД + сервер + клиент)
	@echo "🚀 Запуск полной системы GophKeeper..."
	@echo "📦 База данных: $(shell docker compose ps postgres | grep -q "Up" && echo "✅ Запущена" || echo "❌ Не запущена")"
	@echo "🔧 Сервер: Запускается в фоновом режиме..."
	@echo "💻 Клиент: Запускается..."
	@echo ""
	@echo "📋 Доступные команды в TUI:"
	@echo "   Ctrl+S - Войти/Регистрация"
	@echo "   Ctrl+R - Переключение между входом и регистрацией"
	@echo "   1 - Просмотр данных"
	@echo "   2 - Добавить данные"
	@echo "   3 - Генератор OTP"
	@echo "   s - Синхронизация данных"
	@echo "   q - Выход"
	@echo ""
	@echo "⏳ Ожидание запуска сервера..."
	@sleep 5
	@echo "✅ Система готова! Запускаем клиент..."
	@./bin/client -grpc localhost:9090

.PHONY: vet
vet: ## Проверить код с vet
	go vet -vettool=./statictest-darwin ./...

.PHONY: test-coverage
test-coverage: test ## Показать покрытие тестами
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: proto
proto: ## Генерировать код из proto файлов
	mkdir -p proto/gen
	protoc --go_out=proto/gen --go_opt=paths=source_relative \
		--go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
		proto/*.proto

.PHONY: lint
lint: ## Запустить линтеры
	golangci-lint run

.PHONY: fmt
fmt: ## Форматировать код
	go fmt ./...

.PHONY: clean
clean: ## Очистить артефакты сборки
	rm -rf bin/
	rm -rf proto/gen/
	rm -f coverage.out coverage.html

.PHONY: deps
deps: ## Установить зависимости
	go mod download
	go mod tidy

.PHONY: migrate-up
migrate-up: ## Выполнить миграции вверх
	goose -dir migrations postgres "$(DATABASE_URI)" up

.PHONY: migrate-down
migrate-down: ## Откатить миграции
	goose -dir migrations postgres "$(DATABASE_URI)" down

.PHONY: migrate-status
migrate-status: ## Показать статус миграций
	goose -dir migrations postgres "$(DATABASE_URI)" status

.PHONY: docker-build
docker-build: ## Собрать Docker образы
	docker build -t gophkeeper-server -f Dockerfile.server .
	docker build -t gophkeeper-client -f Dockerfile.client .

.PHONY: docker-run
docker-run: ## Запустить в Docker Compose
	docker-compose up -d

.PHONY: docker-stop
docker-stop: ## Остановить Docker Compose
	docker-compose down

.PHONY: gen-keys
gen-keys: ## Генерировать ключи шифрования
	@mkdir -p keys
	@openssl genrsa -out keys/private.pem 2048
	@openssl rsa -in keys/private.pem -pubout -out keys/public.pem
	@echo "Ключи сгенерированы в папке keys/"

.PHONY: setup
setup: deps gen-keys proto ## Первоначальная настройка проекта
	@echo "Проект настроен! Не забудьте:"
	@echo "1. Создать базу данных PostgreSQL"
	@echo "2. Установить переменную DATABASE_URI"
	@echo "3. Запустить миграции: make migrate-up"

.PHONY: openapi
generate-openapi: ## Генерировать OpenAPI (Swagger) из proto
	protoc -I proto \
		-I /tmp/googleapis \
		--openapiv2_out=proto/gen \
		--openapiv2_opt logtostderr=true \
		--openapiv2_opt allow_merge=true \
		proto/gophkeeper.proto

.DEFAULT_GOAL := help 