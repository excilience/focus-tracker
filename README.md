# Focus Tracker

A simple command-line focus time tracker written in Go.

Focus Tracker helps you track focused work sessions, pause and resume them, check statistics, view session history, and monitor your daily goal.

## What it does

You can use Focus Tracker to:

* start a focus session
* pause and resume the active session
* stop and save a session
* view focus statistics by day, week, month, year, or total
* check daily goal progress
* view session history
* manually edit a session duration

Session data is stored locally in JSON files.


## Installation

Clone the repository:

```bash
git clone https://github.com/excilience/focus-tracker
cd focus-tracker
```
## Settings

Application settings are currently defined in `main.go`:

```go
settings := Settings{
	DayStartHour: 4,
	Timezone:     "",
	DailyGoal:    120,
}
```

## How focus days work

The app uses `DayStartHour` to decide when your focus day starts.

For example:

```go
DayStartHour: 4
```

means that new day starts at **04:00**, not at **00:00**.

So if you work after midnight, for example at `01:30`, that session still belongs to the previous focus day.

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
	Timezone:     "Europe/Istanbul",
	DailyGoal:    120,
}
```

If Timezone is empty:

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

## Project structure

```text
focus-tracker/
├── main.go          # Application startup and command routing
├── commands.go      # CLI command handlers
├── models.go        # Data structures and types
├── session.go       # Start, stop, pause, resume, and edit logic
├── storage.go       # JSON load/save logic
├── format.go        # Time formatting helpers
├── usage.go         # Help messages
├── go.mod
└── data/            # Local JSON storage, created automatically
```

## Run during development

From the project folder:

```bash
go run . start
```

Other examples:

```bash
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

## Commands

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

## Data storage

Focus Tracker stores data locally in JSON files.

The `data/` directory is created automatically after the first command that needs to save data, for example:

```bash
focus start
```
After that, the project will contain:
```text
data/
├── sessions.json
└── active_session.json
```

The directory is created automatically when needed.

`active_session.json` is used while a focus session is running.

`sessions.json` stores finished focus sessions.

You usually should not commit the data/ directory to GitHub, because it contains local user data.


Recommended `.gitignore`:

```gitignore
data/
focus
focus.exe
```

## Notes

Sessions shorter than one minute are not saved.

Paused time is not counted as focused time.

Current storage uses JSON files.

