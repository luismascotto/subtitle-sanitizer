@echo off

cd /d %~dp0

echo Building subtitle-sanitizer...
go build -o subtitle-sanitizer.exe ./cmd/sanitize

if errorlevel 1 (
    echo Failed to build subtitle-sanitizer.
    exit /b 1
)
rem Any parameter means test and build WASM.
rem Learn, as I just did! okay, I'll use a different approach.
rem Using %~1 instead of %1 is safer because it removes any existing double quotes 
rem from the input first, preventing syntax errors if the user provides an argument that already has quotes
rem If your arguments might contain special characters (like &,  , or ^),
rem the bracket method can sometimes fail. A more robust approach is: 
rem if "%~1" equ "" echo Parameter is empty - equ cant take NOT!!! :(
REM FINALLY!

if NOT "%~1" equ "" (
    echo Build OK, now testing...
    go test ./... -count=1 -failfast -coverprofile=coverage.out
    if errorlevel 1 (
        echo Failed to test.
        exit /b 1
    )
    echo Test OK, now building WASM...

    rem call works with powershell? yes it does, but it's not a good idea to call powershell scripts from a batch file.
    rem bullseye!
    pwsh scripts\build-wasm.ps1
)
echo Done!
