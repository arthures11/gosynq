@echo off

echo Starting backend server...
start /B cmd /C "go run cmd\server\main.go"

echo Waiting for backend to start...
timeout /t 2 /nobreak >nul

echo Starting frontend...
start /B cmd /C "cd frontend && ng serve --port 4201"

echo Both backend and frontend are running!
echo Backend should be available at http://localhost:8080
echo Frontend should be available at http://localhost:4201
echo Press Ctrl+C in each terminal to stop services...
echo Or use taskkill /F /IM cmd.exe to kill all cmd processes

pause