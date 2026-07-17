# Focus Tracker

Focus Tracker — это приложение  для учёта времени, проведённого в фокусе, с возможностью отслеживать продуктивность через историю рабочих сессий.

Focus Tracker помогает отслеживать время сфокусированной работы: запускать, ставить на паузу и завершать сессии, смотреть статистику, проверять прогресс по дневной цели, просматривать историю и работать с данными через HTTP API.

## Возможности

Focus Tracker поддерживает:

* запуск фокус-сессии
* паузу и продолжение активной сессии
* завершение и сохранение сессии
* просмотр статистики за день, неделю, месяц, год или за всё время
* проверку прогресса по дневной цели
* просмотр истории сессий
* ручное редактирование длительности сессии
* HTTP API для сессий, статистики, целей и активной сессии
* хранение данных в PostgreSQL
* миграции базы данных через `golang-migrate`
* локальную разработку через Docker Compose и Makefile
* unit-тесты и PostgreSQL integration tests

## Tech stack

* Go
* PostgreSQL
* Docker Compose
* golang-migrate
* net/http

## Установка

Склонируй репозиторий:

```bash
git clone https://github.com/excilience/focus-tracker
cd focus-tracker
```

Установи Go-зависимости:

```bash
go mod download
```

## Конфигурация

Приложение использует переменную окружения `DATABASE_URL` для подключения к PostgreSQL.

Для локальной разработки:

```bash
export DATABASE_URL="postgres://focus:focus@localhost:5433/focus_tracker?sslmode=disable"
```

Если ты используешь готовый Makefile, это значение уже указано там по умолчанию.

Настройки приложения сейчас задаются в `main.go`:

```go
settings := Settings{
	DayStartHour: 4,
	Timezone:     "",
	DailyGoal:    120,
}
```

## Локальная разработка

Запустить PostgreSQL:

```bash
make db-up
```

Применить миграции базы данных:

```bash
make migrate-up
```

Запустить тесты:

```bash
make test
```

Запустить API server:

```bash
make run-api
```

Полностью сбросить локальную базу данных:

```bash
make db-reset
```

Эта команда удаляет Docker volume, заново создаёт PostgreSQL и применяет миграции.

## База данных

Focus Tracker хранит данные в PostgreSQL.

Локальная база данных запускается через Docker Compose.

Схема базы данных управляется миграциями в директории `migrations/`:

```text
migrations/
├── 000001_create_sessions.up.sql
├── 000001_create_sessions.down.sql
├── 000002_create_active_sessions.up.sql
└── 000002_create_active_sessions.down.sql
```

Применить миграции вручную:

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

Откатить одну миграцию:

```bash
migrate -path migrations -database "$DATABASE_URL" down 1
```

Проверить текущую версию миграций:

```bash
migrate -path migrations -database "$DATABASE_URL" version
```

## Запуск во время разработки

Запустить CLI-команду:

```bash
DATABASE_URL="postgres://focus:focus@localhost:5433/focus_tracker?sslmode=disable" go run . start
```

Или используй команды из Makefile, если они доступны.

Примеры:

```bash
go run . start
go run . pause
go run . resume
go run . stop
go run . stats day
go run . stats total
go run . goal
go run . history
```

## Build

Собрать приложение:

```bash
go build -o focus
```

Указать строку подключения к базе данных:

```bash
export DATABASE_URL="postgres://focus:focus@localhost:5433/focus_tracker?sslmode=disable"
```

После этого можно запускать приложение:

```bash
./focus start
./focus stop
./focus stats day
```

На Windows:

```bash
go build -o focus.exe
.\focus.exe start
```

## CLI commands

### Управление сессией

```bash
focus start
focus pause
focus resume
focus stop
```

### Статистика

```bash
focus stats day
focus stats week
focus stats month
focus stats year
focus stats total
```

### Цель и история

```bash
focus goal
focus history
```

### Редактирование сессии

```bash
focus edit <session-id> <duration>
```

Пример:

```bash
focus edit 20260525-143012 1h30m
```

Чтобы найти `session-id`, используй:

```bash
focus history
```

## HTTP API

Запустить API server:

```bash
make run-api
```

Health check:

```bash
curl http://localhost:8080/health
```

### Sessions

Получить список сессий:

```bash
curl http://localhost:8080/sessions
```

Получить сессию по ID:

```bash
curl http://localhost:8080/sessions/<session-id>
```

Запустить сессию:

```bash
curl -X POST http://localhost:8080/sessions/start
```

Поставить активную сессию на паузу:

```bash
curl -X POST http://localhost:8080/sessions/pause
```

Продолжить активную сессию:

```bash
curl -X POST http://localhost:8080/sessions/resume
```

Завершить активную сессию:

```bash
curl -X POST http://localhost:8080/sessions/stop
```

Получить активную сессию:

```bash
curl http://localhost:8080/sessions/active
```

Обновить длительность сессии:

```bash
curl -X PATCH http://localhost:8080/sessions/<session-id> \
  -H "Content-Type: application/json" \
  -d '{"duration":"1h30m"}'
```

Удалить сессию:

```bash
curl -X DELETE http://localhost:8080/sessions/<session-id>
```

Отфильтровать сессии по дате:

```bash
curl "http://localhost:8080/sessions?from=2026-06-01&to=2026-06-30"
```

### Statistics

Получить статистику:

```bash
curl "http://localhost:8080/stats?period=day"
curl "http://localhost:8080/stats?period=week"
curl "http://localhost:8080/stats?period=month"
curl "http://localhost:8080/stats?period=year"
curl "http://localhost:8080/stats?period=total"
```

### Goal

Получить прогресс по дневной цели:

```bash
curl http://localhost:8080/goal
```

## Как работают focus days

Приложение использует настройку `DayStartHour`, чтобы определить, когда начинается твой фокус-день.

Например:

```go
DayStartHour: 4
```

означает, что новый день будет начинаться в **04:00**, а не в **00:00**.

Поэтому если ты работаешь после полуночи, например в `01:30`, эта сессия всё ещё относится к предыдущему фокус-дню.

Это удобно, если твой реальный день часто заканчивается после полуночи.

Пример:

```text
DayStartHour = 4

May 25, 01:30 -> belongs to May 24 focus day
May 25, 05:00 -> belongs to May 25 focus day
```

## Timezone

Приложение использует timezone из настроек:

```go
settings := Settings{
	DayStartHour: 4,
	Timezone:     "Europe/Moscow",
	DailyGoal:    120,
}
```

Если `Timezone` пустой:

```go
Timezone: ""
```

приложение использует локальную timezone твоей системы.

Также можно указать конкретную IANA timezone вручную:

```go
Timezone: "Europe/Moscow"
Timezone: "Asia/Yekaterinburg"
Timezone: "Asia/Dubai"
```

Timezone используется для расчёта фокус-дней, недель, месяцев, лет и дневного прогресса.

PostgreSQL может хранить или отображать timestamps в UTC, но приложение конвертирует время в настроенную timezone при расчёте статистики и форматировании API responses.

## Daily goal

`DailyGoal` указывается в минутах.

Пример:

```go
DailyGoal: 120
```

означает:

```text
Daily goal = 120 minutes = 2 hours
```

Проверить дневной прогресс можно через CLI:

```bash
focus goal
```

или через API:

```bash
curl http://localhost:8080/goal
```

## Tests

Запустить все тесты:

```bash
make test
```

Или вручную:

```bash
TEST_DATABASE_URL="postgres://focus:focus@localhost:5433/focus_tracker?sslmode=disable" go test -v ./...
```

В проекте есть:

* unit tests для логики focus-day и статистики
* HTTP handler tests
* PostgreSQL integration tests

PostgreSQL integration tests требуют запущенную базу данных с применёнными миграциями.

## Project structure

```text
focus-tracker/
├── migrations/                # Database migrations
├── api.go                     # HTTP API handlers and routes
├── commands.go                # CLI command handlers
├── database.go                # Database connection logic
├── docker-compose.yml         # Local PostgreSQL setup
├── errors.go                  # Domain errors
├── executor.go                # Shared DB executor interface
├── focus_service_test.go      # Unit tests for focus and stats logic
├── format.go                  # Time formatting helpers
├── go.mod
├── go.sum
├── main.go                    # Application startup and command routing
├── Makefile                   # Local development commands
├── models.go                  # Data structures and types
├── postgres_storage_test.go   # PostgreSQL integration tests
├── postgres_storage.go        # PostgreSQL storage functions
├── service.go                 # Focus statistics and period calculation logic
├── session.go                 # Start, stop, pause, resume, and edit logic
├── storage.go                 # Legacy JSON storage logic
└── usage.go                   # Help messages
```

## Notes

Сессии короче одной минуты завершаются, но не сохраняются.

Время на паузе не считается сфокусированным временем.

Приложение использует PostgreSQL для постоянного хранения данных.
