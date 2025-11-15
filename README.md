# Pablo Card Game

A multiplayer card game where players try to minimize the sum of their card values.

## Game Rules

- Each player has 4 cards face-down
- Players take turns drawing cards and swapping them with cards in their area
- Special cards:
  - **7**: Look at one of your own cards
  - **8**: Look at someone else's card
  - **9**: Swap any two cards on the table
  - **Red Kings** (hearts/diamonds): Worth -1 to your sum
- Players can call "Pablo" to signal the round is ending
- After Pablo is called and the round ends, everyone reveals and scores
- Lowest total wins!

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

## How to Play

1. Open the game in your browser
2. Enter your name and a game ID (share the same game ID with friends)
3. Wait for at least 2 players to join
4. Start the game
5. Take turns drawing cards and swapping
6. Use special cards strategically
7. Call "Pablo" when you think you have a good hand
8. After the round ends, the player with the lowest score wins!

## Tech Stack

- **Backend**: Go with Gorilla WebSocket
- **Frontend**: Next.js 14 with TypeScript
- **Communication**: WebSocket for real-time multiplayer
