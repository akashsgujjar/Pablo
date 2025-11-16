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
	PabloCaller        string
	StackableCardIndex int    // Index of the last card in discard pile that can be stacked on (placed via end turn, not via stacking)
	StackedSpecialCardPlayers []string // Players who stacked on a special card, waiting for original player to complete
	mu                 sync.RWMutex
}

type Player struct {
	ID    string
	Name  string
	Cards []Card // Changed to slice to support variable number of cards
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
		PabloCaller:        "",
		StackableCardIndex: -1, // -1 means no stackable card
		StackedSpecialCardPlayers: []string{},
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
		Cards: make([]Card, 4),
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
	// Ensure each player has exactly 4 cards
	for playerID := range g.Players {
		// Reset to exactly 4 empty cards first
		g.Players[playerID].Cards = make([]Card, 4)
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

	if g.CurrentPlayer != playerID {
		return false
	}

	// If the deck is empty, automatically end the round and game.
	if len(g.Deck) == 0 {
		// Only end the round if we're still in a playing state
		if g.Status == "playing" {
			g.EndRound()
		}
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

	// Mark this new card as stackable (placed via discard, not via stacking)
	g.StackableCardIndex = len(g.DiscardPile) - 1

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

	if cardIndex < 0 || cardIndex >= len(g.Players[playerID].Cards) {
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

	// Mark this new card as stackable (placed via swap, not via stacking)
	g.StackableCardIndex = len(g.DiscardPile) - 1

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
						if p1, exists1 := g.Players[player1ID]; exists1 && idx1 >= 0 && idx1 < len(p1.Cards) {
							if p2, exists2 := g.Players[player2ID]; exists2 && idx2 >= 0 && idx2 < len(p2.Cards) {
								// Capture card data BEFORE swap for animation
								card1Before := p1.Cards[idx1]
								card2Before := p2.Cards[idx2]
								
								// Broadcast swap event BEFORE swapping so frontend can capture original positions
								g.broadcastSwapEventWithCards(player1ID, idx1, card1Before, player2ID, idx2, card2Before)
								
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
	
	// Check if there are players who stacked on this special card
	// They should get the special card power now
	if len(g.StackedSpecialCardPlayers) > 0 {
		// Get the first player who stacked (FIFO queue)
		stackedPlayerID := g.StackedSpecialCardPlayers[0]
		g.StackedSpecialCardPlayers = g.StackedSpecialCardPlayers[1:]
		
		// Set them as the current player and reactivate the special card
		// This will allow them to use the special card power
		if _, exists := g.Players[stackedPlayerID]; exists {
			g.CurrentPlayer = stackedPlayerID
			g.PendingSpecialCard = cardRank
			g.broadcastGameState()
			return true
		}
	}
	
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
	
	// Check if there are players who stacked on this special card
	// They should get the special card power now
	if len(g.StackedSpecialCardPlayers) > 0 {
		// Get the first player who stacked (FIFO queue)
		stackedPlayerID := g.StackedSpecialCardPlayers[0]
		g.StackedSpecialCardPlayers = g.StackedSpecialCardPlayers[1:]
		
		// Set them as the current player and reactivate the special card
		// This will allow them to use the special card power
		if _, exists := g.Players[stackedPlayerID]; exists {
			g.CurrentPlayer = stackedPlayerID
			// Get the special card rank from the discard pile
			if len(g.DiscardPile) > 0 {
				topCard := g.DiscardPile[len(g.DiscardPile)-1]
				if topCard.Rank == "7" || topCard.Rank == "8" || topCard.Rank == "9" {
					g.PendingSpecialCard = topCard.Rank
				}
			}
			g.broadcastGameState()
			return
		}
	}
	
	g.broadcastGameState()
}

func (g *Game) CallPablo(playerID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Status != "playing" || g.PabloCalled {
		return
	}

	g.PabloCalled = true
	g.PabloCaller = playerID
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

		// If Pablo was called, everyone except the caller gets one more turn.
		// When turn order would come back to the caller, we end the round instead.
		if g.PabloCalled && playerIDs[nextIdx] == g.PabloCaller {
			g.EndRound()
			return
		}

		// Otherwise, pass turn to the next player
		g.CurrentPlayer = playerIDs[nextIdx]
		// Reset the "has drawn" flag for the new current player (fresh turn)
		delete(g.HasDrawnThisTurn, g.CurrentPlayer)
	}

	g.broadcastGameState()
}

func (g *Game) EndRound() {
	g.Status = "ended"
	g.PabloCalled = false
	g.PabloCaller = ""

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

// getNumericRank returns the numeric value of a card rank for comparison
// Returns -1 if the rank doesn't have a numeric value (face cards)
func getNumericRank(rank string) int {
	switch rank {
	case "A":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "5":
		return 5
	case "6":
		return 6
	case "7":
		return 7
	case "8":
		return 8
	case "9":
		return 9
	case "10":
		return 10
	default:
		return -1 // Face cards (J, Q, K) don't have numeric values for stacking
	}
}

// StackCard attempts to stack a player's card on top of the discard pile
// Returns: (success bool, error message string)
func (g *Game) StackCard(playerID string, cardIndex int) (bool, string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if discard pile has a card
	if len(g.DiscardPile) == 0 {
		return false, "No card in discard pile to stack on."
	}

	// Check if the top card is stackable (not placed via stacking)
	// Stacking is allowed if the top card was placed via end turn
	// StackableCardIndex tracks the index of the last card placed via end turn (not via stacking)
	// If StackableCardIndex == -1, it means the top card was placed via stacking, so no stacking allowed
	// If StackableCardIndex != topCardIndex, it means the top card is not the stackable one
	topCardIndex := len(g.DiscardPile) - 1
	
	// Stacking is only allowed if the top card was placed via end turn (not via stacking)
	// This means StackableCardIndex must match topCardIndex
	if g.StackableCardIndex == -1 {
		return false, "Cannot stack on this card. Cards placed via stacking cannot be stacked on."
	}
	if g.StackableCardIndex != topCardIndex {
		return false, "Cannot stack on this card. Only the most recent card placed via end turn can be stacked on."
	}

	// Check if player exists
	player, exists := g.Players[playerID]
	if !exists {
		return false, "Player not found."
	}

	// Check if card index is valid
	if cardIndex < 0 || cardIndex >= len(player.Cards) {
		return false, "Invalid card index."
	}

	// Get the card to stack
	cardToStack := player.Cards[cardIndex]
	if cardToStack.Rank == "" {
		return false, "Invalid card. Card has no rank."
	}

	// Get the top card of discard pile
	topCard := g.DiscardPile[topCardIndex]
	if topCard.Rank == "" {
		return false, "Invalid discard pile card. Card has no rank."
	}

	// Check if ranks match (any rank can stack, including face cards J, Q, K)
	// Suit doesn't matter, only the rank/number needs to match
	if cardToStack.Rank != topCard.Rank {
		// Stack failed - add penalty card
		if len(g.Deck) > 0 {
			penaltyCard := g.Deck[0]
			g.Deck = g.Deck[1:]
			penaltyCard.FaceUp = false
			player.Cards = append(player.Cards, penaltyCard)
		}

		// Immediately broadcast updated game state with penalty card
		g.broadcastGameState()

		// Notify all players about the failed stack attempt
		g.broadcastStackAttempt(playerID, false)

		return false, "Card rank does not match. Penalty card added."
	}

	// Stack successful - remove card from player and add to discard pile
	cardToStack.FaceUp = true
	g.DiscardPile = append(g.DiscardPile, cardToStack)

	// Check if the card being stacked on is a special card (7, 8, 9)
	isStackingOnSpecialCard := topCard.Rank == "7" || topCard.Rank == "8" || topCard.Rank == "9"
	
	// Remove card from player's hand - always remove from slice to make it visible to everyone
	player.Cards = append(player.Cards[:cardIndex], player.Cards[cardIndex+1:]...)
	
	// If we removed one of the first 4 cards and there are penalty cards, move a penalty card up
	// to maintain the visual 4-card structure
	if cardIndex < 4 && len(player.Cards) >= 4 {
		// The slice already has the card removed, so indices have shifted
		// We want to keep the first 4 positions filled if possible
		// Since we removed from index < 4, the card at old index 4 is now at index 3
		// This is already handled by the slice removal above
	}

	// If stacking on a special card, add this player to the queue for special card activation
	if isStackingOnSpecialCard {
		// Only add if not already in the queue
		alreadyQueued := false
		for _, queuedID := range g.StackedSpecialCardPlayers {
			if queuedID == playerID {
				alreadyQueued = true
				break
			}
		}
		if !alreadyQueued {
			g.StackedSpecialCardPlayers = append(g.StackedSpecialCardPlayers, playerID)
		}
	}

	// Mark that the new top card (placed via stacking) cannot be stacked on
	g.StackableCardIndex = -1

	// Notify all players about the successful stack
	g.broadcastStackAttempt(playerID, true)

	g.broadcastGameState()
	return true, ""
}

// broadcastStackAttempt notifies all players about a stack attempt
func (g *Game) broadcastStackAttempt(playerID string, success bool) {
	playerName := ""
	if player, exists := g.Players[playerID]; exists {
		playerName = player.Name
	}

	for _, player := range g.Players {
		if player.Conn != nil {
			message := Message{
				Type: "stackAttempt",
				Payload: map[string]interface{}{
					"playerID":   playerID,
					"playerName": playerName,
					"success":    success,
				},
			}
			player.Conn.WriteJSON(message)
		}
	}
}

// broadcastSwapEventWithCards notifies all players about a card swap with card data for animation
func (g *Game) broadcastSwapEventWithCards(player1ID string, card1Index int, card1 Card, player2ID string, card2Index int, card2 Card) {
	message := Message{
		Type: "swapEvent",
		Payload: map[string]interface{}{
			"player1ID":  player1ID,
			"card1Index": card1Index,
			"card1": map[string]interface{}{
				"suit":   card1.Suit,
				"rank":   card1.Rank,
				"faceUp": card1.FaceUp,
			},
			"player2ID":  player2ID,
			"card2Index": card2Index,
			"card2": map[string]interface{}{
				"suit":   card2.Suit,
				"rank":   card2.Rank,
				"faceUp": card2.FaceUp,
			},
		},
	}

	for _, player := range g.Players {
		if player.Conn != nil {
			player.Conn.WriteJSON(message)
		}
	}
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
		// Filter out empty cards - only include cards with rank or suit
		var cards []Card
		for _, card := range player.Cards {
			if card.Rank != "" || card.Suit != "" {
				// Only show card details if it's the viewer's card, or if it's face up, or if game ended
				if id == viewerID || card.FaceUp || g.Status == "ended" {
					cards = append(cards, Card{
						Suit:   card.Suit,
						Rank:   card.Rank,
						FaceUp: card.FaceUp || g.Status == "ended",
					})
				} else {
					// Hide other players' cards (face down)
					cards = append(cards, Card{
						Suit:   "",
						Rank:   "",
						FaceUp: false,
					})
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

	// Check if stacking is enabled (top card is stackable)
	stackingEnabled := false
	if len(g.DiscardPile) > 0 {
		topCardIndex := len(g.DiscardPile) - 1
		stackingEnabled = g.StackableCardIndex == topCardIndex
	}

	return map[string]interface{}{
		"gameID":             g.ID,
		"players":            players,
		"currentPlayer":     g.CurrentPlayer,
		"status":             g.Status,
		"pabloCalled":        g.PabloCalled,
		"deckSize":           len(g.Deck),
		"discardTop":         getDiscardTop(g.DiscardPile),
		"drawnCards":         drawnCards,
		"pendingSpecialCard": g.PendingSpecialCard,
		"stackingEnabled":    stackingEnabled,
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

		case "stackCard":
			payload := msg.Payload.(map[string]interface{})
			cardIndex := int(payload["cardIndex"].(float64))
			game := gameManager.GetOrCreateGame(gameID)
			success, errorMsg := game.StackCard(playerID, cardIndex)
			if !success {
				// Send error message to the player who attempted to stack
				conn.WriteJSON(Message{
					Type:    "stackError",
					Payload: map[string]string{"message": errorMsg},
				})
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
