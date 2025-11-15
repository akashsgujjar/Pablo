#!/bin/bash

# Start Pablo Card Game
# This script starts both the backend and frontend servers

echo "Starting Pablo Card Game..."

# Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed or not in PATH"
    echo ""
    echo "Please install Go first:"
    echo "  macOS: brew install go"
    echo "  Or download from: https://go.dev/dl/"
    echo ""
    exit 1
fi

# Check for Node.js/npm
if ! command -v npm &> /dev/null; then
    echo "❌ Error: npm is not installed or not in PATH"
    echo ""
    echo "Please install Node.js first:"
    echo "  macOS: brew install node"
    echo "  Or download from: https://nodejs.org/"
    echo ""
    exit 1
fi

# Check if frontend dependencies are installed
if [ ! -d "frontend/node_modules" ]; then
    echo "📦 Installing frontend dependencies..."
    cd frontend
    npm install
    cd ..
fi

# Download Go dependencies
cd backend
echo "📦 Ensuring Go dependencies are up to date..."
go mod download
go mod tidy
cd ..

# Start backend in background
cd backend
echo "Starting Go backend on :8080..."
go run main.go &
BACKEND_PID=$!
cd ..

# Wait a moment for backend to start
sleep 2

# Check if backend started successfully
if ! kill -0 $BACKEND_PID 2>/dev/null; then
    echo "❌ Error: Backend failed to start"
    exit 1
fi

# Start frontend
cd frontend
echo "Starting Next.js frontend on :3000..."
npm run dev &
FRONTEND_PID=$!
cd ..

echo ""
echo "✅ Backend running on http://localhost:8080 (PID: $BACKEND_PID)"
echo "✅ Frontend running on http://localhost:3000 (PID: $FRONTEND_PID)"
echo ""
echo "Press Ctrl+C to stop both servers"

# Wait for user interrupt
trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit" INT TERM
wait

