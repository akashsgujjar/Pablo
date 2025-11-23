# Pablo Card Game

A multiplayer card game where players try to minimize the sum of their card values.

> 📖 **Game Rules**: See [GAME_RULES.md](./GAME_RULES.md) for complete game rules and how to play instructions.

## Prerequisites

Before running the game, you need to install:

1. **Go** (version 1.21 or later)
   - macOS: `brew install go`
   - Or download from: https://go.dev/dl/
   - Verify: `go version`

2. **Node.js** (version 18 or later) and npm
   - macOS: `brew install node`
   - Or download from: https://nodejs.org/
   - Verify: `node --version` and `npm --version`

## Setup

### Quick Start

You can use the provided startup script:

```bash
./start.sh
```

This will:
- Check for required dependencies
- Install frontend dependencies if needed
- Start both the backend and frontend servers

### Manual Setup

#### Backend (Go)

```bash
cd backend
go mod download
go run main.go
```

The backend server runs on `:8080` and handles WebSocket connections.

#### Frontend (Next.js)

In a separate terminal:

```bash
cd frontend
npm install
npm run dev
```

The frontend runs on `http://localhost:3000`


## Tech Stack

- **Backend**: Go with Gorilla WebSocket
- **Frontend**: Next.js 14 with TypeScript
- **Communication**: WebSocket for real-time multiplayer

## TODO
- Cosmetic Fixes
- Fix giving card to someone after correctly stacking theirs
- seating config based on number of players
- refresh persistence
- hosting infra
- test suite