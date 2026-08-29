@echo off
echo Building KOS for Windows...
go build -trimpath -ldflags="-s -w" -o kos.exe main.go
echo.
echo Build complete! You can now run: kos.exe init
echo.
echo To install globally, copy kos.exe to a folder in your PATH, e.g.:
echo   copy kos.exe C:\Windows\
echo.
pause
