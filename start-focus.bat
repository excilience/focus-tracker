@echo off
setlocal EnableExtensions EnableDelayedExpansion
cd /d "%~dp0"

title Focus Tracker

echo.
echo ========================================
echo          Focus Tracker
echo ========================================
echo.

where docker >nul 2>&1
if errorlevel 1 (
    echo Docker Desktop is not installed.
    echo.
    echo Install Docker Desktop, restart Windows,
    echo and run this file again.
    echo.
    start "" "https://www.docker.com/products/docker-desktop/"
    pause
    exit /b 1
)

docker info >nul 2>&1
if not errorlevel 1 goto dockerReady

echo Docker Desktop is installed but not running.
echo Starting Docker Desktop...
echo.

if not exist "%ProgramFiles%\Docker\Docker\Docker Desktop.exe" (
    echo Start Docker Desktop manually and run this file again.
    pause
    exit /b 1
)

start "" "%ProgramFiles%\Docker\Docker\Docker Desktop.exe"

echo Waiting for Docker Desktop...
set /a attempt=0

:waitDocker
timeout /t 3 /nobreak >nul
docker info >nul 2>&1

if not errorlevel 1 goto dockerReady

set /a attempt+=1
echo Waiting for Docker Desktop... !attempt!/40

if !attempt! GEQ 40 (
    echo.
    echo Docker Desktop did not become ready within 2 minutes.
    pause
    exit /b 1
)

goto waitDocker

:dockerReady
echo Docker is ready.
echo.
echo Starting Focus Tracker...
echo.

docker compose up -d --build

if errorlevel 1 (
    echo.
    echo Focus Tracker failed to start.
    echo.
    docker compose ps -a
    echo.
    docker compose logs --tail=80
    echo.
    pause
    exit /b 1
)

echo.
echo Waiting for the application...

powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$url = 'http://localhost:1337/health';" ^
  "for ($i = 0; $i -lt 90; $i++) {" ^
  "  try {" ^
  "    $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 2;" ^
  "    if ($r.StatusCode -eq 200) { exit 0 }" ^
  "  } catch {}" ^
  "  Start-Sleep -Seconds 1" ^
  "}" ^
  "exit 1"

if errorlevel 1 (
    echo.
    echo Focus Tracker did not become ready.
    echo.
    docker compose ps -a
    echo.
    docker compose logs --tail=100
    echo.
    pause
    exit /b 1
)

echo.
echo Focus Tracker is ready.
echo Open in your browser:
echo http://localhost:1337
echo.

timeout /t 3 /nobreak >nul
exit /b 0