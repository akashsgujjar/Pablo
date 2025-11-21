# Pablo Card Game

A multiplayer card game where players try to minimize the sum of their card values.

## Game Rules

### Basic Rules
- **Players**: 2-6 players
- **Starting Hand**: Each player receives 4 face-down cards in a 2x2 grid
- **Turn Order**: Players take turns clockwise
- **Per Turn**: Draw one card, then either discard it or swap it with one of your cards
- **Objective**: Minimize your total card value (lowest score wins)

### Card Values
- **Number cards (2-10)**: Face value
- **Ace**: 1 point
- **Jack, Queen, Black Kings**: 10 points each
- **Red Kings** (♥ hearts, ♦ diamonds): **-1 point** (reduces your score!)

### Special Cards
When placed on the discard pile, these cards activate special powers:
- **7**: Look at one of your own cards
- **8**: Spy on an opponent's face-down card (only you see it)
- **9**: Swap any two cards anywhere on the table

### Stacking Mechanics
- **Stacking**: Match the rank of the top discard card by double-clicking your card or an opponent's card
- **Success**: Card is placed on discard pile, creating an empty slot
- **Failure**: Wrong rank = penalty card added to your hand
- **Stacking on Special Cards**: If you stack on a 7/8/9, you get to use that power after the original player
- **Stacking Opponent's Card**: You must give them one of your cards to replace it

### Ending the Game
- **Call Pablo**: Any player can call "Pablo" - all other players get one more turn, then round ends
- **Deck Empty**: Game auto-ends when deck runs out
- **Zero Cards**: If a player has zero cards, they instantly win!
- **Scoring**: All cards revealed, scores calculated, lowest total wins

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

### Getting Started

1. **Join a Game**: Open the game in your browser and enter your name and a game ID
2. **Share Game ID**: Share the same game ID with friends (2-6 players can join)
3. **Start**: Once at least 2 players have joined, click "Start Game"
4. **Setup**: Each player receives 4 face-down cards arranged in a 2x2 grid

### Turn Structure

On your turn, you can:

1. **Draw a Card**: Click "Draw Card" to take the top card from the deck (you can only draw once per turn)
2. **Choose an Action**:
   - **Discard to Pile**: Place the drawn card face-up on the discard pile
   - **Replace Card**: Swap this card with one of your cards

### Special Cards

When a special card (7, 8, or 9) is placed on the discard pile, click it to activate its power:

- **Card 7**: Look at one of your own cards
  - Click the special card in the discard pile, then click one of your cards to peek at it
  
- **Card 8**: Spy on an opponent's card
  - Click the special card, then click any opponent's face-down card to see it (only you see it!)
  
- **Card 9**: Swap any two cards on the table
  - Click the special card, then click any two cards (yours or opponents') to swap them
  - Watch the animation as cards fly across the table!

**Important**: You must use or skip the special card power before ending your turn.

### Stacking Cards

When a card is placed on the discard pile via discard (not via stacking), you can stack on it:

- **Double-click** any of your cards to attempt stacking
- **Double-click** an opponent's face-down card to attempt stacking their card
- The card rank must **match** the top card on the discard pile
- **Success**: Your card (or their card) is placed on the discard pile, creating an empty slot
- **Failure**: If ranks don't match, you get a penalty card added to your hand

**Stacking on Special Cards**: If you stack on a 7, 8, or 9, you'll be queued to use that special card's power after the original player finishes.

**Stacking Opponent's Cards**: If you successfully stack someone else's card, you must give them one of your cards to replace it. Click one of your cards to give it to them - it goes to the exact slot where their card was.

### Scoring

- **Number cards (2-10)**: Face value (e.g., 5 = 5 points)
- **Ace**: 1 point
- **Jack, Queen, Black Kings**: 10 points each
- **Red Kings** (♥ hearts, ♦ diamonds): **-1 point** (helps your score!)
- **Goal**: Have the **lowest total score** to win

### Winning Conditions

The game ends when:
- Someone calls "Pablo" and all other players have taken one more turn
- The deck runs out (auto-ends when someone tries to draw)
- A player has **zero cards** (instant win!)

### Calling Pablo

- Click "Call Pablo" when you think you have a good (low) hand
- After Pablo is called, each other player gets **one more turn**
- Once the turn order returns to the Pablo caller, the round ends
- All cards are revealed and scores are calculated
- The player with the **lowest score wins**!

## Tech Stack

- **Backend**: Go with Gorilla WebSocket
- **Frontend**: Next.js 14 with TypeScript
- **Communication**: WebSocket for real-time multiplayer
