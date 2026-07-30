@echo off
setlocal
cd /d "%~dp0"

title Stop Focus Tracker

docker compose down

if errorlevel 1 (
    echo.
    echo Failed to stop Focus Tracker.
    pause
    exit /b 1
)

echo.
echo Focus Tracker stopped.
timeout /t 2 /nobreak >nul