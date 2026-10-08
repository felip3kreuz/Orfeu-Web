@echo off
cd /d "%~dp0"
echo Compilando JED v1.3
set GOOS=windows
set CGO_ENABLED=0
set GOARCH=amd64
go build -trimpath -ldflags="-H windowsgui -s -w" -o JED_Simulador.exe .
go build -trimpath -ldflags="-X main.forceClassic=true -s -w" -o JED_Simulador_Classic.exe .
go build -trimpath -ldflags="-X main.diagConsole=true" -o JED_Simulador_Diagnostico.exe .
set GOARCH=386
go build -trimpath -ldflags="-H windowsgui -s -w" -o JED_Simulador_32bits.exe .
if errorlevel 1 goto erro
echo.
echo Build concluido.
pause
exit /b 0
:erro
echo.
echo Falha na compilacao.
pause
exit /b 1
