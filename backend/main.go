package main

import (
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type Game struct {
	ID                 string
	Players            map[string]*Player
	Deck               []Card
	DiscardPile        []Card
	DrawnCards         map[string]*Card // Track drawn card per player
	HasDrawnThisTurn   map[string]bool  // Track if player has drawn this turn
	PendingSpecialCard string           // Track if a special card was just discarded and needs activation
	CurrentPlayer      string
	Status             string // "waiting", "playing", "ended"
	PabloCalled        bool
	mu                 sync.RWMutex
}

type Player struct {
	ID    string
	Name  string
	Cards [4]Card
	Conn  *websocket.Conn
	Ready bool
	Score int
}

type Card struct {
	Suit   string `json:"suit"` // "hearts", "diamonds", "clubs", "spades"
	Rank   string `json:"rank"` // "A", "2", "3", ..., "10", "J", "Q", "K"
	FaceUp bool   `json:"faceUp"`
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func NewGame(id string) *Game {
	game := &Game{
		ID:                 id,
		Players:            make(map[string]*Player),
		Deck:               createDeck(),
		DiscardPile:        []Card{},
		DrawnCards:         make(map[string]*Card),
		HasDrawnThisTurn:   make(map[string]bool),
		PendingSpecialCard: "",
		Status:             "waiting",
		CurrentPlayer:      "",
		PabloCalled:        false,
	}
	shuffleDeck(game.Deck)
	return game
}

func createDeck() []Card {
	suits := []string{"hearts", "diamonds", "clubs", "spades"}
	ranks := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

	deck := make([]Card, 0, 52)
	for _, suit := range suits {
		for _, rank := range ranks {
			deck = append(deck, Card{
				Suit:   suit,
				Rank:   rank,
				FaceUp: false,
			})
		}
	}
	return deck
}

func shuffleDeck(deck []Card) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
}

func (g *Game) AddPlayer(id, name string, conn *websocket.Conn) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.Players) >= 6 {
		return false
	}

	g.Players[id] = &Player{
		ID:    id,
		Name:  name,
		Cards: [4]Card{},
		Conn:  conn,
		Ready: false,
		Score: 0,
	}
	return true
}

func (g *Game) StartGame() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.Players) < 2 {
		return
	}

	g.Status = "playing"

	// Deal 4 cards to each player
	for playerID := range g.Players {
		for i := 0; i < 4; i++ {
			if len(g.Deck) > 0 {
				g.Players[playerID].Cards[i] = g.Deck[0]
				g.Deck = g.Deck[1:]
			}
		}
	}

	// Set first player
	firstPlayer := ""
	for id := range g.Players {
		firstPlayer = id
		break
	}
	g.CurrentPlayer = firstPlayer

	g.broadcastGameState()
}

func (g *Game) DrawCard(playerID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.CurrentPlayer != playerID || len(g.Deck) == 0 {
		return false
	}

	// Can only draw one card per turn - check if they've already drawn this turn
	if g.HasDrawnThisTurn[playerID] {
		return false
	}

	// Draw card and show it to the player
	card := g.Deck[0]
	g.Deck = g.Deck[1:]
	card.FaceUp = true
	g.DrawnCards[playerID] = &card
	g.HasDrawnThisTurn[playerID] = true // Mark that they've drawn this turn

	g.broadcastGameState()
	return true
}

func (g *Game) DiscardDrawnCard(playerID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.CurrentPlayer != playerID {
		return false
	}

	drawnCard, hasDrawnCard := g.DrawnCards[playerID]
	if !hasDrawnCard || drawnCard == nil {
		return false
	}

	// Add drawn card to discard pile (face up so everyone can see)
	card := *drawnCard
	card.FaceUp = true
	g.DiscardPile = append(g.DiscardPile, card)

	// Clear drawn card
	delete(g.DrawnCards, playerID)

	// If it's a special card, mark it as pending activation
	if card.Rank == "7" || card.Rank == "8" || card.Rank == "9" {
		g.PendingSpecialCard = card.Rank
		g.broadcastGameState()
		return true
	}

	// Clear any pending special card if a non-special card was discarded
	g.PendingSpecialCard = ""
	g.broadcastGameState()
	return true
}

func (g *Game) SwapCard(playerID string, cardIndex int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.CurrentPlayer != playerID {
		return false
	}

	drawnCard, hasDrawnCard := g.DrawnCards[playerID]
	if !hasDrawnCard || drawnCard == nil {
		return false
	}

	if cardIndex < 0 || cardIndex >= 4 {
		return false
	}

	// Swap the drawn card with player's card
	oldCard := g.Players[playerID].Cards[cardIndex]
	g.Players[playerID].Cards[cardIndex] = *drawnCard
	g.Players[playerID].Cards[cardIndex].FaceUp = false // Hide it again after swap

	// Add old card to discard pile (face up so everyone can see)
	oldCard.FaceUp = true
	g.DiscardPile = append(g.DiscardPile, oldCard)

	// Clear drawn card
	delete(g.DrawnCards, playerID)

	// If the discarded card is special, mark it as pending activation
	if oldCard.Rank == "7" || oldCard.Rank == "8" || oldCard.Rank == "9" {
		g.PendingSpecialCard = oldCard.Rank
		g.broadcastGameState()
		return true
	}

	// Clear any pending special card if a non-special card was discarded
	g.PendingSpecialCard = ""
	g.broadcastGameState()
	return true
}

// UseSpecialCardFromDiscard is called when a special card is placed in discard pile
func (g *Game) UseSpecialCardFromDiscard(playerID string, cardRank string, params map[string]interface{}) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.CurrentPlayer != playerID {
		return false
	}

	// Check if the top card of discard pile is the special card
	if len(g.DiscardPile) == 0 {
		return false
	}
	topCard := g.DiscardPile[len(g.DiscardPile)-1]
	if topCard.Rank != cardRank {
		return false
	}

	// Also check pending flag for consistency
	if g.PendingSpecialCard != cardRank {
		return false
	}

	switch cardRank {
	case "7": // Look at one of your own cards
		if targetIndex, ok := params["targetIndex"].(float64); ok {
			idx := int(targetIndex)
			if idx >= 0 && idx < 4 {
				card := g.Players[playerID].Cards[idx]
				g.sendToPlayer(playerID, Message{
					Type: "cardRevealed",
					Payload: map[string]interface{}{
						"index": idx,
						"card":  card,
					},
				})
			}
		}

	case "8": // Look at someone else's card
		if targetPlayerID, ok := params["targetPlayerID"].(string); ok {
			if targetIndex, ok2 := params["targetIndex"].(float64); ok2 {
				idx := int(targetIndex)
				if targetPlayer, exists := g.Players[targetPlayerID]; exists && idx >= 0 && idx < 4 {
					card := targetPlayer.Cards[idx]
					g.sendToPlayer(playerID, Message{
						Type: "cardRevealed",
						Payload: map[string]interface{}{
							"playerID": targetPlayerID,
							"index":    idx,
							"card":     card,
						},
					})
				}
			}
		}

	case "9": // Swap any two cards on the table
		if player1ID, ok := params["player1ID"].(string); ok {
			if card1Index, ok2 := params["card1Index"].(float64); ok2 {
				if player2ID, ok3 := params["player2ID"].(string); ok3 {
					if card2Index, ok4 := params["card2Index"].(float64); ok4 {
						idx1 := int(card1Index)
						idx2 := int(card2Index)
						if p1, exists1 := g.Players[player1ID]; exists1 && idx1 >= 0 && idx1 < 4 {
							if p2, exists2 := g.Players[player2ID]; exists2 && idx2 >= 0 && idx2 < 4 {
								// Swap the cards
								p1.Cards[idx1], p2.Cards[idx2] = p2.Cards[idx2], p1.Cards[idx1]
							}
						}
					}
				}
			}
		}
	}

	// Clear the pending special card after use
	g.PendingSpecialCard = ""
	g.broadcastGameState()
	return true
}

func (g *Game) SkipSpecialCard(playerID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.CurrentPlayer != playerID {
		return
	}

	// Clear the pending special card
	g.PendingSpecialCard = ""
	g.broadcastGameState()
}

func (g *Game) CallPablo(playerID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Status != "playing" || g.PabloCalled {
		return
	}

	g.PabloCalled = true
	g.broadcastGameState()
}

func (g *Game) EndTurn(playerID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.CurrentPlayer != playerID {
		return
	}

	// Player must handle drawn card (discard or swap) before ending turn
	if _, hasDrawn := g.DrawnCards[playerID]; hasDrawn {
		return // Can't end turn with a drawn card - must discard or swap first
	}

	// Player must use special card power if one is in the discard pile
	if len(g.DiscardPile) > 0 {
		topCard := g.DiscardPile[len(g.DiscardPile)-1]
		if topCard.Rank == "7" || topCard.Rank == "8" || topCard.Rank == "9" {
			if g.PendingSpecialCard != "" {
				return // Can't end turn with a pending special card - must use it or skip
			}
		}
	}

	// Move to next player
	playerIDs := make([]string, 0, len(g.Players))
	for id := range g.Players {
		playerIDs = append(playerIDs, id)
	}

	currentIdx := -1
	for i, id := range playerIDs {
		if id == playerID {
			currentIdx = i
			break
		}
	}

	if currentIdx >= 0 {
		nextIdx := (currentIdx + 1) % len(playerIDs)
		// Clear any drawn cards from the previous player (safety check)
		delete(g.DrawnCards, playerID)
		// Reset the "has drawn" flag for the previous player
		delete(g.HasDrawnThisTurn, playerID)
		// Reset the "has drawn" flag for the new current player (fresh turn)
		g.CurrentPlayer = playerIDs[nextIdx]
		delete(g.HasDrawnThisTurn, g.CurrentPlayer)
	}

	// If Pablo was called, end the round
	if g.PabloCalled {
		g.EndRound()
		return
	}

	g.broadcastGameState()
}

func (g *Game) EndRound() {
	g.Status = "ended"

	// Reveal all cards
	for _, player := range g.Players {
		for i := range player.Cards {
			player.Cards[i].FaceUp = true
		}
	}

	// Calculate scores
	for _, player := range g.Players {
		score := 0
		for _, card := range player.Cards {
			if card.Rank != "" {
				value := getCardValue(card)
				score += value
			}
		}
		player.Score = score
	}

	g.broadcastGameState()
}

func getCardValue(card Card) int {
	// Red kings (hearts, diamonds) are worth -1
	if card.Rank == "K" && (card.Suit == "hearts" || card.Suit == "diamonds") {
		return -1
	}

	// Face cards
	if card.Rank == "J" || card.Rank == "Q" {
		return 10
	}
	if card.Rank == "K" {
		return 10
	}

	// Ace
	if card.Rank == "A" {
		return 1
	}

	// Number cards
	value := 0
	switch card.Rank {
	case "2":
		value = 2
	case "3":
		value = 3
	case "4":
		value = 4
	case "5":
		value = 5
	case "6":
		value = 6
	case "7":
		value = 7
	case "8":
		value = 8
	case "9":
		value = 9
	case "10":
		value = 10
	}

	return value
}

func (g *Game) sendToPlayer(playerID string, message Message) {
	if player, exists := g.Players[playerID]; exists && player.Conn != nil {
		player.Conn.WriteJSON(message)
	}
}

func (g *Game) broadcastGameState() {
	for playerID, player := range g.Players {
		if player.Conn != nil {
			state := g.getGameStateForPlayer(playerID)
			message := Message{
				Type:    "gameState",
				Payload: state,
			}
			player.Conn.WriteJSON(message)
		}
	}
}

func (g *Game) getGameStateForPlayer(viewerID string) map[string]interface{} {
	players := make(map[string]interface{})
	for id, player := range g.Players {
		cards := make([]Card, 4)
		for i, card := range player.Cards {
			if card.Rank != "" {
				// Only show card details if it's the viewer's card, or if it's face up, or if game ended
				if id == viewerID || card.FaceUp || g.Status == "ended" {
					cards[i] = Card{
						Suit:   card.Suit,
						Rank:   card.Rank,
						FaceUp: card.FaceUp || g.Status == "ended",
					}
				} else {
					// Hide other players' cards
					cards[i] = Card{
						Suit:   "",
						Rank:   "",
						FaceUp: false,
					}
				}
			}
		}
		players[id] = map[string]interface{}{
			"id":    player.ID,
			"name":  player.Name,
			"cards": cards,
			"score": player.Score,
		}
	}

	// Include drawn cards in state (only show your own drawn card)
	drawnCards := make(map[string]*Card)
	if drawnCard, exists := g.DrawnCards[viewerID]; exists && drawnCard != nil {
		drawnCards[viewerID] = drawnCard
	}

	return map[string]interface{}{
		"gameID":             g.ID,
		"players":            players,
		"currentPlayer":      g.CurrentPlayer,
		"status":             g.Status,
		"pabloCalled":        g.PabloCalled,
		"deckSize":           len(g.Deck),
		"discardTop":         getDiscardTop(g.DiscardPile),
		"drawnCards":         drawnCards,
		"pendingSpecialCard": g.PendingSpecialCard,
	}
}

func getDiscardTop(discardPile []Card) *Card {
	if len(discardPile) == 0 {
		return nil
	}
	top := discardPile[len(discardPile)-1]
	return &top
}

type GameManager struct {
	games map[string]*Game
	mu    sync.RWMutex
}

var gameManager = &GameManager{
	games: make(map[string]*Game),
}

func (gm *GameManager) GetOrCreateGame(gameID string) *Game {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if game, exists := gm.games[gameID]; exists {
		return game
	}

	game := NewGame(gameID)
	gm.games[gameID] = game
	return game
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	var playerID, gameID string

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		switch msg.Type {
		case "join":
			payload := msg.Payload.(map[string]interface{})
			gameID = payload["gameID"].(string)
			playerID = payload["playerID"].(string)
			name := payload["name"].(string)

			game := gameManager.GetOrCreateGame(gameID)
			if !game.AddPlayer(playerID, name, conn) {
				conn.WriteJSON(Message{
					Type:    "error",
					Payload: map[string]string{"message": "Game is full"},
				})
				return
			}

			game.broadcastGameState()

		case "startGame":
			game := gameManager.GetOrCreateGame(gameID)
			game.StartGame()

		case "drawCard":
			game := gameManager.GetOrCreateGame(gameID)
			game.DrawCard(playerID)

		case "discardDrawnCard":
			game := gameManager.GetOrCreateGame(gameID)
			game.DiscardDrawnCard(playerID)

		case "swapCard":
			payload := msg.Payload.(map[string]interface{})
			cardIndex := int(payload["cardIndex"].(float64))
			game := gameManager.GetOrCreateGame(gameID)
			game.SwapCard(playerID, cardIndex)

		case "useSpecialCardFromDiscard":
			payload := msg.Payload.(map[string]interface{})
			cardRank := payload["cardRank"].(string)
			params := payload["params"].(map[string]interface{})
			game := gameManager.GetOrCreateGame(gameID)
			game.UseSpecialCardFromDiscard(playerID, cardRank, params)

		case "skipSpecialCard":
			game := gameManager.GetOrCreateGame(gameID)
			game.SkipSpecialCard(playerID)

		case "callPablo":
			game := gameManager.GetOrCreateGame(gameID)
			game.CallPablo(playerID)

		case "endTurn":
			game := gameManager.GetOrCreateGame(gameID)
			game.EndTurn(playerID)
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
