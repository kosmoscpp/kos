@echo off
REM KOS - Zero-Dependency Version Control
REM Build and install for Windows

set BINARY_NAME=kos.exe
set INSTALL_DIR=C:\Program Files\KOS

echo Building KOS...
go build -trimpath -buildvcs=false -ldflags="-s -w" -o %BINARY_NAME% main.go

if %errorlevel% neq 0 (
    echo ❌ Build failed!
    pause
    exit /b 1
)

echo.
echo Installing to %INSTALL_DIR%...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
copy /y %BINARY_NAME% "%INSTALL_DIR%\" >nul

REM Add to PATH (requires admin)
echo Adding to system PATH...
setx PATH "%PATH%;%INSTALL_DIR%" /M >nul 2>&1

echo.
echo ✅ KOS installed successfully!
echo Run 'kos' from anywhere!
echo.
pause
