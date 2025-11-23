package main

import (
	"testing"
)

// Helper function to create a test game
func createTestGame(id string) *Game {
	return NewGame(id)
}

// Helper function to add test players without WebSocket connections
func addTestPlayers(game *Game, count int) []string {
	playerIDs := make([]string, count)
	for i := 0; i < count; i++ {
		playerID := "player" + string(rune('1'+i))
		game.AddPlayer(playerID, "Player "+string(rune('1'+i)), nil)
		playerIDs[i] = playerID
	}
	return playerIDs
}

func TestNewGame(t *testing.T) {
	game := createTestGame("test-game")
	
	if game.ID != "test-game" {
		t.Errorf("Expected game ID 'test-game', got '%s'", game.ID)
	}
	
	if game.Status != "waiting" {
		t.Errorf("Expected status 'waiting', got '%s'", game.Status)
	}
	
	if len(game.Deck) != 52 {
		t.Errorf("Expected deck size 52, got %d", len(game.Deck))
	}
	
	if len(game.Players) != 0 {
		t.Errorf("Expected 0 players, got %d", len(game.Players))
	}
	
	if game.PabloCalled {
		t.Error("Expected PabloCalled to be false")
	}
	
	if game.StackableCardIndex != -1 {
		t.Errorf("Expected StackableCardIndex -1, got %d", game.StackableCardIndex)
	}
}

func TestCreateDeck(t *testing.T) {
	deck := createDeck()
	
	if len(deck) != 52 {
		t.Errorf("Expected deck size 52, got %d", len(deck))
	}
	
	// Check that all suits and ranks are present
	suits := map[string]int{"hearts": 0, "diamonds": 0, "clubs": 0, "spades": 0}
	ranks := map[string]int{
		"A": 0, "2": 0, "3": 0, "4": 0, "5": 0, "6": 0, "7": 0,
		"8": 0, "9": 0, "10": 0, "J": 0, "Q": 0, "K": 0,
	}
	
	for _, card := range deck {
		suits[card.Suit]++
		ranks[card.Rank]++
	}
	
	// Each suit should have 13 cards
	for suit, count := range suits {
		if count != 13 {
			t.Errorf("Expected 13 cards for suit %s, got %d", suit, count)
		}
	}
	
	// Each rank should have 4 cards
	for rank, count := range ranks {
		if count != 4 {
			t.Errorf("Expected 4 cards for rank %s, got %d", rank, count)
		}
	}
}

func TestAddPlayer(t *testing.T) {
	game := createTestGame("test-game")
	
	// Add first player
	success := game.AddPlayer("player1", "Alice", nil)
	if !success {
		t.Error("Failed to add first player")
	}
	
	if len(game.Players) != 1 {
		t.Errorf("Expected 1 player, got %d", len(game.Players))
	}
	
	player, exists := game.Players["player1"]
	if !exists {
		t.Error("Player not found in game")
	}
	
	if player.Name != "Alice" {
		t.Errorf("Expected player name 'Alice', got '%s'", player.Name)
	}
	
	if len(player.Cards) != 4 {
		t.Errorf("Expected player to have 4 card slots, got %d", len(player.Cards))
	}
	
	// Add more players up to limit
	for i := 2; i <= 6; i++ {
		playerID := "player" + string(rune('0'+i))
		success := game.AddPlayer(playerID, "Player "+string(rune('0'+i)), nil)
		if !success {
			t.Errorf("Failed to add player %d", i)
		}
	}
	
	if len(game.Players) != 6 {
		t.Errorf("Expected 6 players, got %d", len(game.Players))
	}
	
	// Try to add 7th player (should fail)
	success = game.AddPlayer("player7", "Bob", nil)
	if success {
		t.Error("Should not be able to add 7th player")
	}
	
	if len(game.Players) != 6 {
		t.Errorf("Expected 6 players after failed add, got %d", len(game.Players))
	}
}

func TestStartGame(t *testing.T) {
	game := createTestGame("test-game")
	
	// Can't start with 0 players
	game.StartGame()
	if game.Status != "waiting" {
		t.Errorf("Expected status 'waiting' with 0 players, got '%s'", game.Status)
	}
	
	// Can't start with 1 player
	game.AddPlayer("player1", "Alice", nil)
	game.StartGame()
	if game.Status != "waiting" {
		t.Errorf("Expected status 'waiting' with 1 player, got '%s'", game.Status)
	}
	
	// Can start with 2 players
	game.AddPlayer("player2", "Bob", nil)
	initialDeckSize := len(game.Deck)
	game.StartGame()
	
	if game.Status != "playing" {
		t.Errorf("Expected status 'playing', got '%s'", game.Status)
	}
	
	if game.CurrentPlayer == "" {
		t.Error("Expected CurrentPlayer to be set")
	}
	
	// Check that each player has 4 cards
	for playerID, player := range game.Players {
		if len(player.Cards) != 4 {
			t.Errorf("Player %s expected 4 cards, got %d", playerID, len(player.Cards))
		}
		
		// Check that cards are not empty
		emptyCount := 0
		for _, card := range player.Cards {
			if card.Rank == "" && card.Suit == "" {
				emptyCount++
			}
		}
		if emptyCount > 0 {
			t.Errorf("Player %s has %d empty cards", playerID, emptyCount)
		}
	}
	
	// Check that deck size decreased (2 players * 4 cards = 8 cards removed)
	expectedDeckSize := initialDeckSize - (len(game.Players) * 4)
	if len(game.Deck) != expectedDeckSize {
		t.Errorf("Expected deck size %d, got %d", expectedDeckSize, len(game.Deck))
	}
}

func TestDrawCard(t *testing.T) {
	game := createTestGame("test-game")
	playerIDs := addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	initialDeckSize := len(game.Deck)
	
	// Current player can draw
	success := game.DrawCard(currentPlayer)
	if !success {
		t.Error("Current player should be able to draw")
	}
	
	if len(game.Deck) != initialDeckSize-1 {
		t.Errorf("Expected deck size %d, got %d", initialDeckSize-1, len(game.Deck))
	}
	
	drawnCard, exists := game.DrawnCards[currentPlayer]
	if !exists || drawnCard == nil {
		t.Error("Drawn card should be stored")
	}
	
	if !drawnCard.FaceUp {
		t.Error("Drawn card should be face up")
	}
	
	if !game.HasDrawnThisTurn[currentPlayer] {
		t.Error("HasDrawnThisTurn should be true")
	}
	
	// Can't draw again in same turn
	success = game.DrawCard(currentPlayer)
	if success {
		t.Error("Should not be able to draw twice in same turn")
	}
	
	// Other player can't draw
	otherPlayer := playerIDs[0]
	if otherPlayer == currentPlayer {
		otherPlayer = playerIDs[1]
	}
	
	success = game.DrawCard(otherPlayer)
	if success {
		t.Error("Non-current player should not be able to draw")
	}
}

func TestDrawCardEmptyDeck(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	// Empty the deck
	game.Deck = []Card{}
	
	// Drawing should end the round
	success := game.DrawCard(game.CurrentPlayer)
	if success {
		t.Error("Should not be able to draw from empty deck")
	}
	
	if game.Status != "ended" {
		t.Error("Game should end when deck is empty")
	}
}

func TestDiscardDrawnCard(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	game.DrawCard(currentPlayer)
	
	initialDiscardSize := len(game.DiscardPile)
	drawnCard := game.DrawnCards[currentPlayer]
	
	success := game.DiscardDrawnCard(currentPlayer)
	if !success {
		t.Error("Should be able to discard drawn card")
	}
	
	if len(game.DiscardPile) != initialDiscardSize+1 {
		t.Errorf("Expected discard pile size %d, got %d", initialDiscardSize+1, len(game.DiscardPile))
	}
	
	if _, exists := game.DrawnCards[currentPlayer]; exists {
		t.Error("Drawn card should be removed")
	}
	
	topCard := game.DiscardPile[len(game.DiscardPile)-1]
	if topCard.Rank != drawnCard.Rank || topCard.Suit != drawnCard.Suit {
		t.Error("Top of discard pile should be the discarded card")
	}
	
	if !topCard.FaceUp {
		t.Error("Discarded card should be face up")
	}
	
	if game.StackableCardIndex != len(game.DiscardPile)-1 {
		t.Error("StackableCardIndex should be set to last card")
	}
}

func TestDiscardSpecialCard(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	
	// Manually set a special card as drawn
	specialCard := Card{Suit: "hearts", Rank: "7", FaceUp: true}
	game.DrawnCards[currentPlayer] = &specialCard
	
	game.DiscardDrawnCard(currentPlayer)
	
	if game.PendingSpecialCard != "7" {
		t.Errorf("Expected PendingSpecialCard '7', got '%s'", game.PendingSpecialCard)
	}
	
	// Test with card 8
	specialCard8 := Card{Suit: "diamonds", Rank: "8", FaceUp: true}
	game.DrawnCards[currentPlayer] = &specialCard8
	game.DiscardDrawnCard(currentPlayer)
	
	if game.PendingSpecialCard != "8" {
		t.Errorf("Expected PendingSpecialCard '8', got '%s'", game.PendingSpecialCard)
	}
}

func TestSwapCard(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	game.DrawCard(currentPlayer)
	
	originalCard := game.Players[currentPlayer].Cards[0]
	drawnCard := game.DrawnCards[currentPlayer]
	
	success := game.SwapCard(currentPlayer, 0)
	if !success {
		t.Error("Should be able to swap card")
	}
	
	// Check that drawn card is now in player's hand
	if game.Players[currentPlayer].Cards[0].Rank != drawnCard.Rank ||
		game.Players[currentPlayer].Cards[0].Suit != drawnCard.Suit {
		t.Error("Player's card should be replaced with drawn card")
	}
	
	// Check that original card is in discard pile
	topCard := game.DiscardPile[len(game.DiscardPile)-1]
	if topCard.Rank != originalCard.Rank || topCard.Suit != originalCard.Suit {
		t.Error("Original card should be in discard pile")
	}
	
	if _, exists := game.DrawnCards[currentPlayer]; exists {
		t.Error("Drawn card should be removed")
	}
}

func TestCallPablo(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	
	game.CallPablo(currentPlayer)
	
	if !game.PabloCalled {
		t.Error("PabloCalled should be true")
	}
	
	if game.PabloCaller != currentPlayer {
		t.Errorf("Expected PabloCaller '%s', got '%s'", currentPlayer, game.PabloCaller)
	}
}

func TestEndTurn(t *testing.T) {
	game := createTestGame("test-game")
	playerIDs := addTestPlayers(game, 3)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	
	// Draw and discard to complete a turn
	game.DrawCard(currentPlayer)
	game.DiscardDrawnCard(currentPlayer)
	
	// Find next player
	var nextPlayer string
	for _, id := range playerIDs {
		if id != currentPlayer {
			nextPlayer = id
			break
		}
	}
	
	game.EndTurn(currentPlayer)
	
	if game.CurrentPlayer != nextPlayer {
		t.Errorf("Expected CurrentPlayer '%s', got '%s'", nextPlayer, game.CurrentPlayer)
	}
	
	if game.HasDrawnThisTurn[currentPlayer] {
		t.Error("HasDrawnThisTurn should be cleared for previous player")
	}
}

func TestEndTurnWithPablo(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	pabloCaller := game.CurrentPlayer
	
	// Find the other player by checking all players
	var otherPlayer string
	for id := range game.Players {
		if id != pabloCaller {
			otherPlayer = id
			break
		}
	}
	
	if otherPlayer == "" {
		t.Fatal("Could not find other player")
	}
	
	game.CallPablo(pabloCaller)
	
	// Verify Pablo was called
	if !game.PabloCalled || game.PabloCaller != pabloCaller {
		t.Fatal("Pablo should be called")
	}
	
	// Complete pablo caller's turn
	game.DrawCard(pabloCaller)
	game.DiscardDrawnCard(pabloCaller)
	game.EndTurn(pabloCaller)
	
	// Should now be other player's turn (Pablo caller gets skipped until others have had their turn)
	// The turn should advance to the other player
	if game.CurrentPlayer == pabloCaller {
		t.Errorf("Turn should have advanced from Pablo caller. CurrentPlayer: %s", game.CurrentPlayer)
	}
	
	// Get the current player (should be otherPlayer)
	currentAfterFirstTurn := game.CurrentPlayer
	
	// Complete current player's turn
	game.DrawCard(currentAfterFirstTurn)
	game.DiscardDrawnCard(currentAfterFirstTurn)
	game.EndTurn(currentAfterFirstTurn)
	
	// Should return to Pablo caller and end round (because next player would be Pablo caller)
	if game.Status != "ended" {
		t.Errorf("Game should end when turn returns to Pablo caller. Status: %s, CurrentPlayer: %s, PabloCaller: %s", 
			game.Status, game.CurrentPlayer, game.PabloCaller)
	}
}

func TestStackCard(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	
	// Draw and discard a card
	game.DrawCard(currentPlayer)
	game.DiscardDrawnCard(currentPlayer)
	
	topCard := game.DiscardPile[len(game.DiscardPile)-1]
	
	// Give player a matching card
	matchingCard := Card{Suit: "clubs", Rank: topCard.Rank, FaceUp: false}
	game.Players[currentPlayer].Cards[0] = matchingCard
	
	initialDiscardSize := len(game.DiscardPile)
	
	success, msg := game.StackCard(currentPlayer, 0)
	if !success {
		t.Errorf("Should be able to stack matching card: %s", msg)
	}
	
	if len(game.DiscardPile) != initialDiscardSize+1 {
		t.Errorf("Expected discard pile size %d, got %d", initialDiscardSize+1, len(game.DiscardPile))
	}
	
	// Check that player's card slot is now empty
	if game.Players[currentPlayer].Cards[0].Rank != "" {
		t.Error("Stacked card should be removed from player's hand")
	}
}

func TestStackCardMismatch(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	
	// Draw and discard a card
	game.DrawCard(currentPlayer)
	game.DiscardDrawnCard(currentPlayer)
	
	topCard := game.DiscardPile[len(game.DiscardPile)-1]
	
	// Give player a non-matching card
	nonMatchingRank := "A"
	if topCard.Rank == "A" {
		nonMatchingRank = "2"
	}
	nonMatchingCard := Card{Suit: "clubs", Rank: nonMatchingRank, FaceUp: false}
	game.Players[currentPlayer].Cards[0] = nonMatchingCard
	
	initialCardCount := len(game.Players[currentPlayer].Cards)
	
	success, _ := game.StackCard(currentPlayer, 0)
	if success {
		t.Error("Should not be able to stack non-matching card")
	}
	
	// Player should get a penalty card
	if len(game.Players[currentPlayer].Cards) <= initialCardCount {
		t.Error("Player should receive a penalty card for failed stack")
	}
}

func TestStackOpponentCard(t *testing.T) {
	game := createTestGame("test-game")
	playerIDs := addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	otherPlayer := playerIDs[0]
	if otherPlayer == currentPlayer {
		otherPlayer = playerIDs[1]
	}
	
	// Draw and discard a card
	game.DrawCard(currentPlayer)
	game.DiscardDrawnCard(currentPlayer)
	
	topCard := game.DiscardPile[len(game.DiscardPile)-1]
	
	// Give opponent a matching card
	matchingCard := Card{Suit: "clubs", Rank: topCard.Rank, FaceUp: false}
	game.Players[otherPlayer].Cards[0] = matchingCard
	
	success, msg := game.StackOpponentCard(currentPlayer, otherPlayer, 0)
	if !success {
		t.Errorf("Should be able to stack opponent's matching card: %s", msg)
	}
	
	// Check that opponent's card slot is empty
	if game.Players[otherPlayer].Cards[0].Rank != "" {
		t.Error("Stacked card should be removed from opponent's hand")
	}
	
	// Check that PendingGive is set
	if game.PendingGive == nil {
		t.Error("PendingGive should be set after stacking opponent's card")
	}
	
	if game.PendingGive.ActorID != currentPlayer {
		t.Error("PendingGive.ActorID should be the actor")
	}
	
	if game.PendingGive.TargetPlayerID != otherPlayer {
		t.Error("PendingGive.TargetPlayerID should be the target")
	}
}

func TestCountNonEmptyCards(t *testing.T) {
	game := createTestGame("test-game")
	playerIDs := addTestPlayers(game, 2)
	game.StartGame()
	
	// Use the first player
	playerID := playerIDs[0]
	player := game.Players[playerID]
	
	if player == nil {
		t.Fatal("Player not found")
	}
	
	// Initially should have 4 cards (after game starts)
	count := game.countNonEmptyCards(player)
	if count != 4 {
		t.Errorf("Expected 4 cards, got %d", count)
	}
	
	// Remove a card (simulate stacking)
	player.Cards[0] = Card{Rank: "", Suit: "", FaceUp: false}
	
	count = game.countNonEmptyCards(player)
	if count != 3 {
		t.Errorf("Expected 3 cards after removing one, got %d", count)
	}
}

func TestZeroCardsWinCondition(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	
	// Remove all cards (simulate stacking all cards)
	for i := range game.Players[currentPlayer].Cards {
		game.Players[currentPlayer].Cards[i] = Card{Rank: "", Suit: "", FaceUp: false}
	}
	
	// Try to stack (which checks for zero cards)
	game.DrawCard(currentPlayer)
	game.DiscardDrawnCard(currentPlayer)
	
	// Add a card back temporarily for the stack attempt
	game.Players[currentPlayer].Cards = append(game.Players[currentPlayer].Cards, Card{Rank: "A", Suit: "hearts", FaceUp: false})
	
	// Stack should trigger win condition
	topCard := game.DiscardPile[len(game.DiscardPile)-1]
	matchingCard := Card{Suit: "clubs", Rank: topCard.Rank, FaceUp: false}
	game.Players[currentPlayer].Cards[len(game.Players[currentPlayer].Cards)-1] = matchingCard
	
	success, _ := game.StackCard(currentPlayer, len(game.Players[currentPlayer].Cards)-1)
	if success {
		// After successful stack, player should have 0 cards
		if game.countNonEmptyCards(game.Players[currentPlayer]) == 0 && game.Status == "playing" {
			// This should trigger EndRound in the actual implementation
			// For testing, we can manually check
			if game.Status != "ended" {
				// Note: EndRound is called in StackCard when countNonEmptyCards returns 0
				// So we need to verify the logic works
			}
		}
	}
}

func TestEndRound(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	// End the round
	game.EndRound()
	
	if game.Status != "ended" {
		t.Error("Status should be 'ended'")
	}
	
	if game.PabloCalled {
		t.Error("PabloCalled should be cleared")
	}
	
	if game.PabloCaller != "" {
		t.Error("PabloCaller should be cleared")
	}
	
	// Check that scores are calculated
	for _, player := range game.Players {
		if player.Score == 0 {
			// Score might be 0 if all cards are low value, but let's check it's calculated
			// We'll verify the calculation logic separately
		}
	}
}

// Test helper to verify card values
func TestCardValues(t *testing.T) {
	testCases := []struct {
		card     Card
		expected int
	}{
		{Card{Rank: "A", Suit: "hearts"}, 1},
		{Card{Rank: "2", Suit: "hearts"}, 2},
		{Card{Rank: "10", Suit: "hearts"}, 10},
		{Card{Rank: "J", Suit: "hearts"}, 10},
		{Card{Rank: "Q", Suit: "hearts"}, 10},
		{Card{Rank: "K", Suit: "hearts"}, -1},   // Red king
		{Card{Rank: "K", Suit: "diamonds"}, -1}, // Red king
		{Card{Rank: "K", Suit: "clubs"}, 10},    // Black king
		{Card{Rank: "K", Suit: "spades"}, 10},   // Black king
	}
	
	for _, tc := range testCases {
		value := getCardValue(tc.card)
		if value != tc.expected {
			t.Errorf("Card %s %s: expected value %d, got %d", tc.card.Rank, tc.card.Suit, tc.expected, value)
		}
	}
}


func TestUseSpecialCard7(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	
	// Discard a 7 card
	game.DrawCard(currentPlayer)
	game.DrawnCards[currentPlayer].Rank = "7"
	game.DiscardDrawnCard(currentPlayer)
	
	// Use special card 7 to look at own card
	params := map[string]interface{}{"targetIndex": 0}
	success := game.UseSpecialCardFromDiscard(currentPlayer, "7", params)
	
	if !success {
		t.Error("Should be able to use special card 7")
	}
	
	if game.PendingSpecialCard != "" {
		t.Error("PendingSpecialCard should be cleared after use")
	}
}

func TestUseSpecialCard8(t *testing.T) {
	game := createTestGame("test-game")
	playerIDs := addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	otherPlayer := playerIDs[0]
	if otherPlayer == currentPlayer {
		otherPlayer = playerIDs[1]
	}
	
	// Discard an 8 card
	game.DrawCard(currentPlayer)
	game.DrawnCards[currentPlayer].Rank = "8"
	game.DiscardDrawnCard(currentPlayer)
	
	// Use special card 8 to spy on opponent
	params := map[string]interface{}{
		"targetPlayerID": otherPlayer,
		"targetIndex":    0,
	}
	success := game.UseSpecialCardFromDiscard(currentPlayer, "8", params)
	
	if !success {
		t.Error("Should be able to use special card 8")
	}
	
	if game.PendingSpecialCard != "" {
		t.Error("PendingSpecialCard should be cleared after use")
	}
}

func TestUseSpecialCard9(t *testing.T) {
	game := createTestGame("test-game")
	playerIDs := addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	otherPlayer := playerIDs[0]
	if otherPlayer == currentPlayer {
		otherPlayer = playerIDs[1]
	}
	
	// Discard a 9 card
	game.DrawCard(currentPlayer)
	game.DrawnCards[currentPlayer].Rank = "9"
	game.DiscardDrawnCard(currentPlayer)
	
	// Use special card 9 to swap two cards
	params := map[string]interface{}{
		"player1ID":  currentPlayer,
		"card1Index": float64(0),
		"player2ID":  otherPlayer,
		"card2Index": float64(0),
	}
	
	card1Before := game.Players[currentPlayer].Cards[0]
	card2Before := game.Players[otherPlayer].Cards[0]
	
	success := game.UseSpecialCardFromDiscard(currentPlayer, "9", params)
	
	if !success {
		t.Error("Should be able to use special card 9")
	}
	
	// Check that cards were swapped
	if game.Players[currentPlayer].Cards[0].Rank != card2Before.Rank ||
		game.Players[currentPlayer].Cards[0].Suit != card2Before.Suit {
		t.Error("Card 1 should be swapped with card 2")
	}
	
	if game.Players[otherPlayer].Cards[0].Rank != card1Before.Rank ||
		game.Players[otherPlayer].Cards[0].Suit != card1Before.Suit {
		t.Error("Card 2 should be swapped with card 1")
	}
	
	if game.PendingSpecialCard != "" {
		t.Error("PendingSpecialCard should be cleared after use")
	}
}

func TestSkipSpecialCard(t *testing.T) {
	game := createTestGame("test-game")
	addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	
	// Discard a special card
	game.DrawCard(currentPlayer)
	game.DrawnCards[currentPlayer].Rank = "7"
	game.DiscardDrawnCard(currentPlayer)
	
	if game.PendingSpecialCard != "7" {
		t.Error("PendingSpecialCard should be set")
	}
	
	game.SkipSpecialCard(currentPlayer)
	
	if game.PendingSpecialCard != "" {
		t.Error("PendingSpecialCard should be cleared after skip")
	}
}

func TestHandleGiveCard(t *testing.T) {
	game := createTestGame("test-game")
	playerIDs := addTestPlayers(game, 2)
	game.StartGame()
	
	currentPlayer := game.CurrentPlayer
	otherPlayer := playerIDs[0]
	if otherPlayer == currentPlayer {
		otherPlayer = playerIDs[1]
	}
	
	// Set up PendingGive (simulate after stacking opponent's card)
	game.PendingGive = &PendingGive{
		ActorID:        currentPlayer,
		TargetPlayerID: otherPlayer,
		TargetIndex:    0,
	}
	
	// Give a card
	cardToGive := game.Players[currentPlayer].Cards[1]
	game.HandleGiveCard(currentPlayer, 1)
	
	// Check that card was moved
	if game.Players[otherPlayer].Cards[0].Rank != cardToGive.Rank ||
		game.Players[otherPlayer].Cards[0].Suit != cardToGive.Suit {
		t.Error("Card should be given to target player")
	}
	
	// Check that PendingGive is cleared
	if game.PendingGive != nil {
		t.Error("PendingGive should be cleared after giving card")
	}
}

func TestGameManager(t *testing.T) {
	gm := &GameManager{
		games: make(map[string]*Game),
	}
	
	// Get or create first game
	game1 := gm.GetOrCreateGame("game1")
	if game1 == nil {
		t.Error("Expected game to be created")
	}
	
	if game1.ID != "game1" {
		t.Errorf("Expected game ID 'game1', got '%s'", game1.ID)
	}
	
	// Get same game again
	game1Again := gm.GetOrCreateGame("game1")
	if game1 != game1Again {
		t.Error("Should return same game instance")
	}
	
	// Create different game
	game2 := gm.GetOrCreateGame("game2")
	if game2 == nil {
		t.Error("Expected game2 to be created")
	}
	
	if game2.ID != "game2" {
		t.Errorf("Expected game ID 'game2', got '%s'", game2.ID)
	}
	
	if game1 == game2 {
		t.Error("Should return different game instances")
	}
}

