@echo off
rem Сборка Pyatimirye.exe на Windows. Нужен Go 1.25+ (https://go.dev/dl/).
cd /d "%~dp0\.."
go test ./... || exit /b 1
go build -trimpath -ldflags "-s -w -H=windowsgui" -o dist\Pyatimirye.exe .\cmd\pyatimirye || exit /b 1
echo Готово: dist\Pyatimirye.exe
