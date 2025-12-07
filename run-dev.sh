#!/bin/bash

# Start backend in background
echo "Starting backend server..."
go run cmd/server/main.go &

# Get the backend process ID
BACKEND_PID=$!

# Give backend a moment to start
sleep 2

# Start frontend in background
echo "Starting frontend..."
cd frontend || exit
ng serve --port 4201 &

# Get the frontend process ID
FRONTEND_PID=$!

echo "Both backend and frontend are running!"
echo "Backend should be available at http://localhost:8080"
echo "Frontend should be available at http://localhost:4201"
echo "Backend PID: $BACKEND_PID"
echo "Frontend PID: $FRONTEND_PID"
echo "Press Ctrl+C to stop both services..."

# Wait for user to stop
trap "kill $BACKEND_PID $FRONTEND_PID; exit" INT

# Keep the script running
while true; do
    sleep 1
done