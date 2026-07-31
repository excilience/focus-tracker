@echo off
setlocal
cd /d "%~dp0"

title Focus Tracker Logs

docker compose logs -f --tail=200

pause