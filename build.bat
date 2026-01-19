@echo off

cd /d %~dp0

echo Building subtitle-sanitizer...
go build -o subtitle-sanitizer.exe ./cmd/sanitize

if errorlevel 1 (
    echo Failed to build subtitle-sanitizer.
    exit /b 1
)
echo Done!
