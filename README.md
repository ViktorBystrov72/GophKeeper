# GophKeeper - Менеджер паролей

Клиент-серверная система для надежного и безопасного хранения логинов, паролей, бинарных данных и прочей приватной информации.

## Возможности

- 🔐 **Безопасное хранение**: Шифрование данных с помощью RSA + AES
- 👤 **Аутентификация**: JWT токены с поддержкой обновления
- 🔑 **OTP**: Поддержка одноразовых паролей (TOTP)
- 🌐 **Бинарный протокол**: gRPC для эффективного взаимодействия
- 💻 **Кроссплатформенность**: CLI клиент для Windows, Linux, macOS
- 🎨 **TUI интерфейс**: Удобный терминальный интерфейс
- 🔄 **Синхронизация**: Автоматическая синхронизация между клиентами
- 📦 **Типы данных**: Логины/пароли, текст, бинарные данные, банковские карты
- 📋 **Просмотр данных**: Получение и просмотр приватных данных владельцем

## Архитектура

Проект состоит из:
- **Сервер**: HTTP/gRPC API с PostgreSQL
- **Клиент**: CLI приложение с TUI интерфейсом

## Синхронизация данных

Система поддерживает полную синхронизацию данных между несколькими авторизованными клиентами одного владельца:

### Автоматическая синхронизация:
- ✅ При входе в систему
- ✅ При регистрации нового пользователя
- ✅ При переходе к списку данных

### Ручная синхронизация:
- ✅ Клавиша `s` в главном меню для принудительной синхронизации

### Отслеживание изменений:
- ✅ Измененные записи после времени последней синхронизации
- ✅ Удаленные записи с сохранением истории
- ✅ Оптимистичное блокирование с версионированием

### Интерфейс синхронизации:
- ✅ Отображение времени последней синхронизации
- ✅ Статус синхронизации с количеством обновленных/удаленных записей
- ✅ Интеграция с TUI для удобного использования

## Передача приватных данных владельцу

Система обеспечивает безопасную передачу приватных данных владельцу по запросу:

### Серверная часть:
- ✅ **gRPC метод `GetData`** - получение конкретной записи по ID
- ✅ **gRPC метод `ListData`** - получение списка всех записей пользователя
- ✅ **HTTP endpoints** - `/data/{id}` и `/data` для REST API
- ✅ **Аутентификация и авторизация** - проверка JWT токенов
- ✅ **Проверка владельца** - пользователь может получить только свои данные
- ✅ **Шифрование данных** - данные хранятся в зашифрованном виде

### Клиентская часть (TUI):
- ✅ **Просмотр списка данных** - клавиша '1' в главном меню
- ✅ **Выбор записи** - Enter для просмотра конкретной записи
- ✅ **Детальный просмотр** - отображение названия, описания, ID записи
- ✅ **Удаление записи** - Delete в режиме просмотра
- ✅ **Навигация** - Esc для возврата к списку

## Требования

- Go 1.24.4+
- PostgreSQL 13+
- protoc (для сборки gRPC)
- Docker (для запуска базы данных)

## Быстрый старт

### 1. Запуск базы данных
```bash
docker compose up -d postgres
```

### 2. Запуск сервера
```bash
./scripts/start-server.sh
```

### 3. Запуск клиента
```bash
./bin/client -grpc localhost:9090
```

### 4. Полный запуск с UI (рекомендуется)
```bash
make ui
```

Эта команда автоматически:
- Запустит базу данных PostgreSQL
- Запустит сервер в фоновом режиме
- Запустит клиент с TUI интерфейсом

### Альтернативный запуск сервера
```bash
DATABASE_URI="postgres://gophkeeper:password@localhost:5432/gophkeeper?sslmode=disable" \
JWT_SECRET="your-secret-key" \
ENCRYPTION_KEY="your-encryption-key" \
LOG_LEVEL="info" \
SERVER_ADDRESS=":8080" \
GRPC_ADDRESS=":9090" \
./bin/server
```

## Быстрый старт

### 1. Настройка проекта

```bash
# Клонирование и настройка
git clone <repository-url>
cd gophkeeper
make setup
```

### 2. Настройка базы данных

```bash
# Создайте базу данных PostgreSQL
createdb gophkeeper

# Установите переменную окружения
export DATABASE_URI="postgres://user:password@localhost:5432/gophkeeper?sslmode=disable"

# Выполните миграции
make migrate-up
```

### 3. Запуск сервера

```bash
# Установите секретные ключи
export JWT_SECRET="your-jwt-secret-key"
export ENCRYPTION_KEY="your-encryption-key"

# Запустите сервер
make run-server
```

### 4. Запуск клиента

```bash
make run-client
```

## Использование TUI интерфейса

GophKeeper предоставляет интерактивный терминальный интерфейс (TUI) для удобной работы с данными.

### Запуск клиента

```bash
# Запуск интерактивного режима
./bin/client

# Показать версию
./bin/client version

# Показать справку
./bin/client help
```

### Навигация по интерфейсу

#### 🔐 Экран входа

При первом запуске вы увидите экран входа:

```
🔐 GophKeeper - Вход в систему

Имя пользователя:
[Введите имя пользователя]

Пароль:
[••••••••]

Ctrl+S: Войти • Ctrl+R: Регистрация • Ctrl+C: Выход
```

**Управление:**
- `Tab` / `Shift+Tab` - переключение между полями
- `Ctrl+S` - войти в систему
- `Ctrl+R` - перейти к регистрации
- `Ctrl+C` - выйти из приложения

#### 📝 Экран регистрации

Если у вас нет аккаунта, нажмите `Ctrl+R` для регистрации:

```
📝 GophKeeper - Регистрация

Имя пользователя:
[Введите имя пользователя]

Пароль:
[••••••••]

Ctrl+S: Зарегистрироваться • Ctrl+L: Вход • Ctrl+C: Выход
```

**Управление:**
- `Tab` / `Shift+Tab` - переключение между полями
- `Ctrl+S` - зарегистрироваться
- `Ctrl+L` - вернуться к входу
- `Ctrl+C` - выйти из приложения

#### 🏠 Главное меню

После успешной аутентификации откроется главное меню:

```
🏠 GophKeeper - Добро пожаловать, username!

Выберите действие:

1. 📋 Просмотр данных
2. ➕ Добавить данные
3. 🔑 Генератор OTP
q. ❌ Выход

Используйте цифры для выбора • q: Выход
```

**Управление:**
- `1` - просмотр сохраненных данных
- `2` - добавление новых данных
- `3` - генератор OTP кодов
- `q` - выйти из приложения

#### 📋 Просмотр данных

В списке данных вы можете:

```
GophKeeper - Менеджер паролей

┌─ GitHub Account ──────────────────────────────┐
│ Логин: user@example.com                      │
│ Тип: credentials                             │
│ Обновлено: 2024-01-15 10:30:00              │
└───────────────────────────────────────────────┘

┌─ Банковская карта ───────────────────────────┐
│ Номер: **** **** **** 1234                   │
│ Тип: card                                     │
│ Обновлено: 2024-01-10 14:20:00              │
└───────────────────────────────────────────────┘

↑/↓: Навигация • Enter: Просмотр • Delete: Удалить • Esc: Назад
```

**Управление:**
- `↑` / `↓` - навигация по списку
- `Enter` - просмотр выбранной записи
- `Delete` - удалить выбранную запись
- `Esc` - вернуться в главное меню

#### 🔑 Генератор OTP

Для создания OTP секретов и генерации кодов:

```
🔑 Генерация OTP

Имя аккаунта (email/login):
[Введите имя аккаунта]

Секрет: JBSWY3DPEHPK3PXP
QR: otpauth://totp/GophKeeper:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=GophKeeper
Резервные коды:
  12345678
  87654321
  11223344
  55667788

Текущий OTP: 123456

Ctrl+G: сгенерировать OTP • Esc: назад
```

**Управление:**
- `Ctrl+S` - создать новый OTP секрет
- `Ctrl+G` - сгенерировать текущий OTP код
- `Esc` - вернуться в главное меню

### Типы данных

GophKeeper поддерживает следующие типы данных:

#### 🔐 Логины и пароли
- Логин и пароль
- URL сайта
- Дополнительные заметки

#### 📝 Текстовые данные
- Произвольный текст
- Заметки и описание

#### 📄 Бинарные данные
- Файлы любого формата
- Автоматическое определение MIME-типа
- Безопасное хранение в зашифрованном виде

#### 💳 Банковские карты
- Номер карты
- Срок действия
- Имя держателя
- CVV код
- PIN код (опционально)
- Заметки

### Безопасность

- 🔒 **Шифрование**: Все данные шифруются перед сохранением
- 🔐 **Пароли**: Скрываются при вводе (••••••••)
- 🎫 **JWT токены**: Автоматическое обновление токенов
- 🔑 **OTP**: Дополнительная защита через одноразовые пароли
- 🛡️ **Валидация**: Проверка всех входных данных

### Горячие клавиши

| Клавиша | Действие |
|---------|----------|
| `Tab` / `Shift+Tab` | Переключение между полями |
| `Enter` | Подтверждение действия |
| `Esc` | Отмена / Назад |
| `Ctrl+C` | Выход из приложения |
| `Ctrl+S` | Сохранить / Войти / Создать |
| `Ctrl+R` | Регистрация |
| `Ctrl+L` | Вход в систему |
| `Ctrl+G` | Генерировать OTP |
| `↑` / `↓` | Навигация по списку |
| `Delete` | Удалить элемент |

### Советы по использованию

1. **Первый запуск**: Зарегистрируйтесь с надежным паролем
2. **Регулярная синхронизация**: Данные автоматически синхронизируются с сервером
3. **OTP коды**: Используйте для дополнительной защиты важных аккаунтов
4. **Резервные коды**: Сохраните резервные коды OTP в безопасном месте
5. **Безопасный выход**: Всегда используйте `Ctrl+C` для корректного завершения

## Разработка

### Доступные команды

```bash
make help                 # Показать все доступные команды
make build                # Собрать сервер и клиент
make test                 # Запустить тесты
make test-coverage        # Показать покрытие тестами
make proto               # Генерировать код из proto файлов
make lint                # Запустить линтеры
make fmt                 # Форматировать код
```

### Docker

```bash
# Запуск в Docker
make docker-build
make docker-run

# Остановка
make docker-stop
```

## Конфигурация

### Сервер

Конфигурация через флаги командной строки или переменные окружения:

| Флаг | Переменная | Описание | По умолчанию |
|------|------------|----------|--------------|
| `-a` | `SERVER_ADDRESS` | Адрес HTTP сервера | `:8080` |
| `-g` | `GRPC_ADDRESS` | Адрес gRPC сервера | `:8081` |
| `-d` | `DATABASE_URI` | Строка подключения к БД | **обязательно** |
| `-jwt` | `JWT_SECRET` | Секретный ключ для JWT | **обязательно** |
| `-enc` | `ENCRYPTION_KEY` | Ключ шифрования | **обязательно** |
| `-s` | `ENABLE_TLS` | Включить TLS | `false` |
| `-l` | `LOG_LEVEL` | Уровень логирования | `info` |

### Клиент

| Флаг | Переменная | Описание | По умолчанию |
|------|------------|----------|--------------|
| `-server` | `SERVER_ADDRESS` | Адрес сервера | `localhost:8080` |
| `-grpc` | `GRPC_ADDRESS` | Адрес gRPC сервера | `localhost:8081` |
| `-tls` | `ENABLE_TLS` | Использовать TLS | `false` |
| `-config` | `CONFIG_PATH` | Путь к файлу конфигурации | `./config.json` |

## API

### REST API

- `POST /auth/register` - Регистрация
- `POST /auth/login` - Аутентификация
- `POST /auth/refresh` - Обновление токена
- `GET /data` - Список данных
- `POST /data` - Создание данных
- `GET /data/{id}` - Получение данных
- `PUT /data/{id}` - Обновление данных
- `DELETE /data/{id}` - Удаление данных
- `POST /sync` - Синхронизация
- `POST /otp/generate` - Генерация OTP
- `POST /otp/secret` - Создание OTP секрета

### gRPC API

См. `proto/gophkeeper.proto` для полного описания gRPC интерфейса.

## Типы данных

### Логины/пароли
```json
{
  "login": "username",
  "password": "password",
  "url": "https://example.com",
  "notes": "дополнительная информация"
}
```

### Текстовые данные
```json
{
  "content": "секретный текст",
  "notes": "описание"
}
```

### Бинарные данные
```json
{
  "filename": "document.pdf",
  "content": "base64-encoded-data",
  "mime_type": "application/pdf",
  "notes": "важный документ"
}
```

### Банковские карты
```json
{
  "number": "1234567890123456",
  "expiry_date": "12/25",
  "holder": "IVAN PETROV",
  "cvv": "123",
  "pin": "1234",
  "notes": "основная карта"
}
```

## Безопасность

- Все данные шифруются перед сохранением в базе
- Используется комбинация RSA + AES для оптимального шифрования
- Пароли хешируются с помощью bcrypt
- JWT токены с ограниченным временем жизни
- Поддержка OTP для дополнительной безопасности

## Тестирование

```bash
# Запуск всех тестов
make test

# Покрытие тестами
make test-coverage
```

Цель покрытия: **80%+**

## Лицензия

MIT

## Разработка

Проект разработан с учетом лучших практик Go:
- Использование современных библиотек (pgx, zap, chi)
- Структурированное логирование
- Graceful shutdown
- Dependency injection
- Тестируемый код с интерфейсами
- Комплексная обработка ошибок

## Swagger/OpenAPI

Полная документация API доступна в формате OpenAPI (Swagger) 2.0.

### Генерация документации

```bash
# Генерация Swagger документации
make generate-openapi
```

### Просмотр документации

Файл документации: `proto/gen/gophkeeper.swagger.json`

**Способы просмотра:**

1. **Swagger Editor** (онлайн): https://editor.swagger.io/
   - Загрузите файл `proto/gen/gophkeeper.swagger.json`

2. **Swagger UI** (локально):
   ```bash
   # Установка Swagger UI
   docker run -p 8080:8080 -e SWAGGER_JSON=/swagger.json -v $(pwd)/proto/gen/gophkeeper.swagger.json:/swagger.json swaggerapi/swagger-ui
   ```
   - Откройте http://localhost:8080

3. **Redoc** (альтернативный UI):
   ```bash
   # Установка Redoc
   npm install -g redoc-cli
   redoc-cli serve proto/gen/gophkeeper.swagger.json
   ```

### Описание API

Документация включает:

#### 🔐 **Аутентификация**
- `POST /auth/register` - Регистрация пользователя
- `POST /auth/login` - Вход в систему
- `POST /auth/refresh` - Обновление JWT токена

#### 📊 **Управление данными**
- `GET /data` - Получение списка данных (с фильтрацией и пагинацией)
- `POST /data` - Создание новой записи данных
- `GET /data/{id}` - Получение конкретной записи
- `PUT /data/{id}` - Обновление записи данных
- `DELETE /data/{id}` - Удаление записи данных
- `POST /sync` - Синхронизация данных между клиентами

#### 🔐 **OTP функциональность**
- `POST /otp/generate` - Генерация одноразового пароля
- `POST /otp/secret` - Создание нового OTP секрета

#### 🔒 **Безопасность**
- JWT токены для аутентификации
- Bearer токены в заголовке Authorization
- Шифрование всех данных
- Поддержка CORS

#### 📋 **Типы данных**
- `DATA_TYPE_CREDENTIALS` - Логины/пароли
- `DATA_TYPE_TEXT` - Текстовые данные
- `DATA_TYPE_BINARY` - Бинарные данные
- `DATA_TYPE_CARD` - Банковские карты

### Примеры запросов

#### Регистрация пользователя
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'
```

#### Создание записи данных
```bash
curl -X POST http://localhost:8080/data \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "type": "DATA_TYPE_CREDENTIALS",
    "name": "GitHub",
    "description": "GitHub аккаунт",
    "encryptedData": "encrypted_data_here",
    "metadata": "github.com"
  }'
```

#### Получение списка данных
```bash
curl -X GET "http://localhost:8080/data?type=DATA_TYPE_CREDENTIALS&limit=10&offset=0" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```
