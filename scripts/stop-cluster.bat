@echo off
REM Stop DistKV development cluster

echo Stopping DistKV development cluster...

REM Kill all distkv-server.exe processes
taskkill /F /IM distkv-server.exe 2>nul
if %errorlevel% equ 0 (
    echo Development cluster stopped successfully.
) else (
    echo No running DistKV nodes found.
)

echo.
pause