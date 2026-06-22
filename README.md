# Focus Tracker

A simple focus time tracker written in Go.

Focus Tracker helps you track focused work sessions, pause and resume them, check statistics, view session history, monitor your daily goal, and access the data through an HTTP API.

## Features

Focus Tracker supports:

* starting a focus session
* pausing and resuming the active session
* stopping and saving a session
* viewing focus statistics by day, week, month, year, or total
* checking daily goal progress
* viewing session history
* manually editing a session duration
* HTTP API endpoints for sessions, statistics, goals, and active session state
* PostgreSQL storage
* database migrations with `golang-migrate`
* local development workflow with Docker Compose and Makefile
* unit and PostgreSQL integration tests

## Tech stack

* Go
* PostgreSQL
* Docker Compose
* golang-migrate
* net/http

## Installation

Clone the repository:

```bash
git clone https://github.com/excilience/focus-tracker
cd focus-tracker
```

Install Go dependencies:

```bash
go mod download
```

## Configuration

The app uses the `DATABASE_URL` environment variable to connect to PostgreSQL.

For local development:

```bash
export DATABASE_URL="postgres://focus:focus@localhost:5433/focus_tracker?sslmode=disable"
```

If you use the provided Makefile, this value is already defined there for local development.

Application settings are currently defined in `main.go`:

```go
settings := Settings{
	DayStartHour: 4,
	Timezone:     "",
	DailyGoal:    120,
}
```

## Local development

Start PostgreSQL:

```bash
make db-up
```

Apply database migrations:

```bash
make migrate-up
```

Run tests:

```bash
make test
```

Run the API server:

```bash
make run-api
```

Reset the local database completely:

```bash
make db-reset
```

This removes the Docker volume, recreates PostgreSQL, and applies migrations again.

## Database

Focus Tracker stores data in PostgreSQL.

The local database is started with Docker Compose.

Database schema is managed by migrations in the `migrations/` directory:

```text
migrations/
├── 000001_create_sessions.up.sql
├── 000001_create_sessions.down.sql
├── 000002_create_active_sessions.up.sql
└── 000002_create_active_sessions.down.sql
```

Apply migrations manually:

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

Rollback one migration:

```bash
migrate -path migrations -database "$DATABASE_URL" down 1
```

Check current migration version:

```bash
migrate -path migrations -database "$DATABASE_URL" version
```

## Run during development

Run a CLI command:

```bash
DATABASE_URL="postgres://focus:focus@localhost:5433/focus_tracker?sslmode=disable" go run . start
```

Or use Makefile commands where available.

Examples:

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

Build the app:

```bash
go build -o focus
```

Set the database connection string:

```bash
export DATABASE_URL="postgres://focus:focus@localhost:5433/focus_tracker?sslmode=disable"
```

Then run it:

```bash
./focus start
./focus stop
./focus stats day
```

On Windows:

```bash
go build -o focus.exe
.\focus.exe start
```

## CLI commands

### Control

```bash
focus start
focus pause
focus resume
focus stop
```

### Statistics

```bash
focus stats day
focus stats week
focus stats month
focus stats year
focus stats total
```

### Goal and history

```bash
focus goal
focus history
```

### Edit a session

```bash
focus edit <session-id> <duration>
```

Example:

```bash
focus edit 20260525-143012 1h30m
```

To find a session ID:

```bash
focus history
```

## HTTP API

Start the API server:

```bash
make run-api
```

Health check:

```bash
curl http://localhost:8080/health
```

### Sessions

List sessions:

```bash
curl http://localhost:8080/sessions
```

Get a session by ID:

```bash
curl http://localhost:8080/sessions/<session-id>
```

Start a session:

```bash
curl -X POST http://localhost:8080/sessions/start
```

Pause the active session:

```bash
curl -X POST http://localhost:8080/sessions/pause
```

Resume the active session:

```bash
curl -X POST http://localhost:8080/sessions/resume
```

Stop the active session:

```bash
curl -X POST http://localhost:8080/sessions/stop
```

Get the active session:

```bash
curl http://localhost:8080/sessions/active
```

Update a session duration:

```bash
curl -X PATCH http://localhost:8080/sessions/<session-id> \
  -H "Content-Type: application/json" \
  -d '{"duration":"1h30m"}'
```

Delete a session:

```bash
curl -X DELETE http://localhost:8080/sessions/<session-id>
```

Filter sessions by date:

```bash
curl "http://localhost:8080/sessions?from=2026-06-01&to=2026-06-30"
```

### Statistics

Get statistics:

```bash
curl "http://localhost:8080/stats?period=day"
curl "http://localhost:8080/stats?period=week"
curl "http://localhost:8080/stats?period=month"
curl "http://localhost:8080/stats?period=year"
curl "http://localhost:8080/stats?period=total"
```

### Goal

Get daily goal progress:

```bash
curl http://localhost:8080/goal
```

## How focus days work

The app uses `DayStartHour` to decide when your focus day starts.

For example:

```go
DayStartHour: 4
```

means that a new focus day starts at **04:00**, not at **00:00**.

So if you work after midnight, for example at `01:30`, that session still belongs to the previous day.

This is useful if your real day often ends after midnight.

Example:

```text
DayStartHour = 4

May 25, 01:30 -> belongs to May 24 focus day
May 25, 05:00 -> belongs to May 25 focus day
```

## Timezone

The app uses the timezone from settings:

```go
settings := Settings{
	DayStartHour: 4,
	Timezone:     "Europe/Moscow",
	DailyGoal:    120,
}
```

If `Timezone` is empty:

```go
Timezone: ""
```

the app uses your local system timezone.

You can also set a specific IANA timezone manually:

```go
Timezone: "Europe/Moscow"
Timezone: "Asia/Yekaterinburg"
Timezone: "Asia/Dubai"
```

The timezone is used for calculating focus days, weeks, months, years, and daily progress.

PostgreSQL may store or display timestamps in UTC, but the app converts time to the configured timezone when calculating stats and formatting API responses.

## Daily goal

`DailyGoal` is written in minutes.

Example:

```go
DailyGoal: 120
```

means:

```text
Daily goal = 120 minutes = 2 hours
```

You can check your daily progress with:

```bash
focus goal
```

or through the API:

```bash
curl http://localhost:8080/goal
```

## Tests

Run all tests:

```bash
make test
```

Or manually:

```bash
TEST_DATABASE_URL="postgres://focus:focus@localhost:5433/focus_tracker?sslmode=disable" go test -v ./...
```

The project includes:

* unit tests for focus-day and statistics logic
* HTTP handler tests
* PostgreSQL integration tests

PostgreSQL integration tests require a running database with migrations applied.

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

Sessions shorter than one minute are stopped but not saved.

Paused time is not counted as focused time.

The app uses PostgreSQL for persistent storage.

Database structure is managed through versioned migrations.
