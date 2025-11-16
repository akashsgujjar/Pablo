'use client'

import { useState, useEffect, useRef } from 'react'
import styles from './page.module.css'

interface Card {
  suit: string
  rank: string
  faceUp: boolean
}

interface Player {
  id: string
  name: string
  cards: Card[]
  score: number
}

interface GameState {
  gameID: string
  players: { [key: string]: Player }
  currentPlayer: string
  status: string
  pabloCalled: boolean
  deckSize: number
  discardTop: Card | null
  drawnCards: { [key: string]: Card }
  pendingSpecialCard: string
}

export default function Home() {
  const [gameID, setGameID] = useState('')
  const [playerID, setPlayerID] = useState('')
  const [playerName, setPlayerName] = useState('')
  const [connected, setConnected] = useState(false)
  const [gameState, setGameState] = useState<GameState | null>(null)
  const [selectedCardIndex, setSelectedCardIndex] = useState<number | null>(null)
  const [drawnCard, setDrawnCard] = useState<Card | null>(null)
  const [isDrawing, setIsDrawing] = useState(false)
  const [revealedCard, setRevealedCard] = useState<{ playerID: string; index: number; card: Card } | null>(null)
  const [swapSelection, setSwapSelection] = useState<{ playerID: string; cardIndex: number } | null>(null)
  const [selectedSpecialCard, setSelectedSpecialCard] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    // Generate player ID if not set
    if (!playerID) {
      setPlayerID(`player_${Math.random().toString(36).substr(2, 9)}`)
    }
  }, [playerID])

  const connectWebSocket = () => {
    if (!gameID || !playerName) {
      alert('Please enter game ID and your name')
      return
    }

    const ws = new WebSocket('ws://localhost:8080/ws')
    wsRef.current = ws

    ws.onopen = () => {
      setConnected(true)
      ws.send(JSON.stringify({
        type: 'join',
        payload: {
          gameID,
          playerID,
          name: playerName,
        },
      }))
    }

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data)
      
      if (message.type === 'gameState') {
        const state = message.payload
        setGameState(state)
        
        const shouldClearSpecial =
          state.currentPlayer !== playerID ||
          !state.pendingSpecialCard ||
          !state.discardTop ||
          state.discardTop.rank !== state.pendingSpecialCard

        if (state.currentPlayer !== playerID) {
          setIsDrawing(false)
        }

        if (shouldClearSpecial) {
          setSelectedSpecialCard(null)
        }
        
        // Update drawn card if it exists for this player
        if (state.drawnCards && state.drawnCards[playerID]) {
          setDrawnCard(state.drawnCards[playerID])
          setIsDrawing(false) // Reset drawing flag when we get the actual card
        } else {
          setDrawnCard(null)
          setIsDrawing(false) // Reset drawing flag
        }
      } else if (message.type === 'cardRevealed') {
        setRevealedCard(message.payload)
        setTimeout(() => setRevealedCard(null), 3000)
      } else if (message.type === 'error') {
        alert(message.payload.message)
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    ws.onclose = () => {
      setConnected(false)
    }
  }

  const sendMessage = (type: string, payload: any) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type, payload }))
    }
  }

  const handleStartGame = () => {
    sendMessage('startGame', {})
  }

  const handleDrawCard = () => {
    // Prevent multiple draws
    if (drawnCard || isDrawing) {
      return
    }
    setIsDrawing(true)
    sendMessage('drawCard', {})
  }

  const handleDiscardDrawnCard = () => {
    sendMessage('discardDrawnCard', {})
    setDrawnCard(null)
  }

  const handleSwapCard = (cardIndex: number) => {
    if (drawnCard) {
      sendMessage('swapCard', { cardIndex })
      setDrawnCard(null)
      setSelectedCardIndex(null)
    }
  }

  const handleUseSpecialCardFromDiscard = (cardRank: string, params: any) => {
    sendMessage('useSpecialCardFromDiscard', { cardRank, params })
  }

  const handleCallPablo = () => {
    sendMessage('callPablo', {})
  }

  const handleEndTurn = () => {
    sendMessage('endTurn', {})
  }

  const getCardValue = (card: Card): number => {
    if (!card || !card.rank) return 0
    if (card.rank === 'K' && (card.suit === 'hearts' || card.suit === 'diamonds')) {
      return -1
    }
    if (card.rank === 'A') return 1
    if (card.rank === 'J' || card.rank === 'Q' || card.rank === 'K') return 10
    return parseInt(card.rank) || 0
  }

  const isRedSuit = (suit?: string): boolean => {
    return suit === 'hearts' || suit === 'diamonds'
  }

  const getPlayerTotal = (player: Player): number => {
    return player.cards.reduce((sum, card) => sum + getCardValue(card), 0)
  }

  const isMyTurn = gameState?.currentPlayer === playerID
  const myPlayer = gameState?.players[playerID]
  const otherPlayers = gameState ? Object.values(gameState.players).filter(p => p.id !== playerID) : []

  if (!connected) {
    return (
      <div className={styles.container}>
        <div className={styles.joinForm}>
          <h1>🎴 Pablo</h1>
          <input
            type="text"
            placeholder="Your Name"
            value={playerName}
            onChange={(e) => setPlayerName(e.target.value)}
            className={styles.input}
          />
          <input
            type="text"
            placeholder="Game ID"
            value={gameID}
            onChange={(e) => setGameID(e.target.value)}
            className={styles.input}
          />
          <button onClick={connectWebSocket} className={styles.button}>
            Join Game
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <div className={styles.gameHeader}>
        <h1>🎴 Pablo</h1>
        <div className={styles.gameInfo}>
          <span>Game: {gameID}</span>
          <span>Status: {gameState?.status}</span>
          {gameState?.pabloCalled && <span className={styles.pabloCalled}>PABLO CALLED!</span>}
        </div>
      </div>

      {gameState?.status === 'waiting' && (
        <div className={styles.waitingRoom}>
          <h2>Waiting for players...</h2>
          <div className={styles.playerList}>
            {Object.values(gameState.players).map((player) => (
              <div key={player.id} className={styles.playerCard}>
                {player.name}
              </div>
            ))}
          </div>
          {Object.keys(gameState.players).length >= 2 && (
            <button onClick={handleStartGame} className={styles.button}>
              Start Game
            </button>
          )}
        </div>
      )}

      {gameState?.status === 'playing' && (
        <>
          {/* Discard Pile and Special Card UI */}
          <div className={styles.discardPileContainer}>
            {gameState.discardTop && (
              <div className={styles.discardPile}>
                <h3>Discard Pile</h3>
                <div
                  className={`${styles.card} ${
                    isMyTurn &&
                    gameState.discardTop &&
                    gameState.pendingSpecialCard === gameState.discardTop.rank
                      ? styles.clickableCard
                      : ''
                  }`}
                  onClick={() => {
                    console.log('Discard pile card clicked', {
                      isMyTurn,
                      discardTop: gameState.discardTop,
                      rank: gameState.discardTop?.rank,
                      pendingSpecial: gameState.pendingSpecialCard,
                    })
                    if (
                      isMyTurn &&
                      gameState.discardTop &&
                      gameState.pendingSpecialCard === gameState.discardTop.rank
                    ) {
                      console.log('Setting selectedSpecialCard to', gameState.discardTop.rank)
                      setSelectedSpecialCard(gameState.discardTop.rank)
                    }
                  }}
                  style={{
                    cursor:
                      isMyTurn &&
                      gameState.discardTop &&
                      gameState.pendingSpecialCard === gameState.discardTop.rank
                        ? 'pointer'
                        : 'default',
                    border:
                      isMyTurn &&
                      gameState.discardTop &&
                      gameState.pendingSpecialCard === gameState.discardTop.rank
                        ? '3px solid #ffd700'
                        : 'none',
                  }}
                >
                  <div className={styles.cardFace}>
                    <span className={styles.rank}>{gameState.discardTop.rank}</span>
                    <span
                      className={`${styles.suit} ${
                        isRedSuit(gameState.discardTop.suit) ? styles.redSuit : styles.blackSuit
                      }`}
                    >
                      {getSuitSymbol(gameState.discardTop.suit)}
                    </span>
                    {isMyTurn &&
                      gameState.discardTop &&
                      gameState.pendingSpecialCard === gameState.discardTop.rank && (
                      <div style={{ fontSize: '10px', marginTop: '5px', color: '#ffd700', fontWeight: 'bold' }}>✨ CLICK ✨</div>
                    )}
                  </div>
                </div>
              </div>
            )}

            {/* Special Card Actions - Shown when user clicks on special card in discard pile */}
            {isMyTurn &&
              selectedSpecialCard &&
              gameState?.discardTop &&
              gameState.pendingSpecialCard === selectedSpecialCard &&
              gameState.discardTop.rank === selectedSpecialCard && (
              <div className={styles.specialCardMenu}>
                <h3>✨ Special Card Power: {selectedSpecialCard} ✨</h3>
                <button onClick={() => setSelectedSpecialCard(null)} className={styles.button} style={{ marginBottom: '10px' }}>
                  Close
                </button>
                {selectedSpecialCard === '7' && (
                  <div>
                    <h4>Look at one of your cards:</h4>
                    {[0, 1, 2, 3].map((idx) => (
                      <button
                        key={idx}
                      onClick={() => {
                        handleUseSpecialCardFromDiscard('7', { targetIndex: idx })
                        setSelectedSpecialCard(null)
                      }}
                        className={styles.button}
                      >
                        Card {idx + 1}
                      </button>
                    ))}
                  </div>
                )}
                {selectedSpecialCard === '8' && (
                  <div>
                    <h4>Look at someone else's card:</h4>
                    {otherPlayers.map((player) => (
                      <div key={player.id}>
                        <h5>{player.name}</h5>
                        {[0, 1, 2, 3].map((idx) => (
                          <button
                            key={idx}
                            onClick={() => {
                              handleUseSpecialCardFromDiscard('8', {
                                targetPlayerID: player.id,
                                targetIndex: idx,
                              })
                              setSelectedSpecialCard(null)
                            }}
                            className={styles.button}
                          >
                            Card {idx + 1}
                          </button>
                        ))}
                      </div>
                    ))}
                  </div>
                )}
                {selectedSpecialCard === '9' && (
                  <div>
                    <h4>Swap any two cards:</h4>
                    {!swapSelection ? (
                      <div>
                        <p>Select the first card to swap:</p>
                        <div>
                          <h5>Your cards:</h5>
                          {[0, 1, 2, 3].map((idx) => (
                            <button
                              key={idx}
                              onClick={() => setSwapSelection({ playerID: playerID, cardIndex: idx })}
                              className={styles.button}
                            >
                              Your Card {idx + 1}
                            </button>
                          ))}
                        </div>
                        {otherPlayers.map((player) => (
                          <div key={player.id}>
                            <h5>{player.name}'s cards:</h5>
                            {[0, 1, 2, 3].map((idx) => (
                              <button
                                key={idx}
                                onClick={() => setSwapSelection({ playerID: player.id, cardIndex: idx })}
                                className={styles.button}
                              >
                                {player.name}'s Card {idx + 1}
                              </button>
                            ))}
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div>
                        <p>First card selected. Now select the second card:</p>
                        <div>
                          <h5>Your cards:</h5>
                          {[0, 1, 2, 3].map((idx) => {
                            const isSelected = swapSelection.playerID === playerID && swapSelection.cardIndex === idx
                            return (
                              <button
                                key={idx}
                                onClick={() => {
                                  if (!isSelected) {
                                    handleUseSpecialCardFromDiscard('9', {
                                      player1ID: swapSelection.playerID,
                                      card1Index: swapSelection.cardIndex,
                                      player2ID: playerID,
                                      card2Index: idx,
                                    })
                                    setSwapSelection(null)
                                    setSelectedSpecialCard(null)
                                  }
                                }}
                                className={styles.button}
                                disabled={isSelected}
                              >
                                Your Card {idx + 1}
                              </button>
                            )
                          })}
                        </div>
                        {otherPlayers.map((player) => (
                          <div key={player.id}>
                            <h5>{player.name}'s cards:</h5>
                            {[0, 1, 2, 3].map((idx) => {
                              const isSelected = swapSelection.playerID === player.id && swapSelection.cardIndex === idx
                              return (
                                <button
                                  key={idx}
                                  onClick={() => {
                                    if (!isSelected) {
                                      handleUseSpecialCardFromDiscard('9', {
                                        player1ID: swapSelection.playerID,
                                        card1Index: swapSelection.cardIndex,
                                        player2ID: player.id,
                                        card2Index: idx,
                                      })
                                      setSwapSelection(null)
                                      setSelectedSpecialCard(null)
                                    }
                                  }}
                                  className={styles.button}
                                  disabled={isSelected}
                                >
                                  {player.name}'s Card {idx + 1}
                                </button>
                              )
                            })}
                          </div>
                        ))}
                        <button onClick={() => setSwapSelection(null)} className={styles.button}>
                          Cancel Selection
                        </button>
                      </div>
                    )}
                  </div>
                )}
                <button onClick={() => {
                  sendMessage('skipSpecialCard', {})
                  setSwapSelection(null)
                  setSelectedSpecialCard(null)
                }} className={styles.button}>
                  Skip Special Card
                </button>
              </div>
            )}
          </div>

          {/* Other Players */}
          <div className={styles.otherPlayers}>
            {otherPlayers.map((player) => (
              <div key={player.id} className={styles.playerArea}>
                <h3>{player.name} {gameState.currentPlayer === player.id && '👈'}</h3>
                <div className={styles.cardRow}>
                  {player.cards.map((card, idx) => (
                    <div
                      key={idx}
                      className={`${styles.card} ${styles.cardBack}`}
                    >
                      {card.faceUp && card.rank ? (
                        <div className={styles.cardFace}>
                          <span className={styles.rank}>{card.rank}</span>
                          <span
                            className={`${styles.suit} ${
                              isRedSuit(card.suit) ? styles.redSuit : styles.blackSuit
                            }`}
                          >
                            {getSuitSymbol(card.suit)}
                          </span>
                        </div>
                      ) : (
                        <div className={styles.cardBackPattern}>🎴</div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>

          {/* My Cards */}
          {myPlayer && (
            <div className={styles.myArea}>
              <h2>Your Cards {isMyTurn && '👈 Your Turn'}</h2>
              <div className={styles.cardRow}>
                {myPlayer.cards.map((card, idx) => {
                  const isSpecial = card.rank === '7' || card.rank === '8' || card.rank === '9'
                  return (
                    <div
                      key={idx}
                      className={`${styles.card} ${card.faceUp ? styles.cardFace : styles.cardBack} ${selectedCardIndex === idx ? styles.selected : ''}`}
                      onClick={() => {
                        if (isMyTurn && drawnCard) {
                          handleSwapCard(idx)
                        }
                      }}
                    >
                      {card.faceUp || card.rank ? (
                        <div className={styles.cardFace}>
                          <span className={styles.rank}>{card.rank}</span>
                          <span
                            className={`${styles.suit} ${
                              isRedSuit(card.suit) ? styles.redSuit : styles.blackSuit
                            }`}
                          >
                            {getSuitSymbol(card.suit)}
                          </span>
                          {isSpecial && <span className={styles.specialBadge}>✨</span>}
                        </div>
                      ) : (
                        <div className={styles.cardBackPattern}>🎴</div>
                      )}
                    </div>
                  )
                })}
              </div>
              <div className={styles.myTotal}>
                Total: {getPlayerTotal(myPlayer)}
              </div>
            </div>
          )}


          {/* Revealed Card Modal */}
          {revealedCard && (
            <div className={styles.modal}>
              <div className={styles.modalContent}>
                <h3>Revealed Card</h3>
                <div className={styles.card}>
                  <div className={styles.cardFace}>
                    <span className={styles.rank}>{revealedCard.card.rank}</span>
                    <span
                      className={`${styles.suit} ${
                        isRedSuit(revealedCard.card.suit) ? styles.redSuit : styles.blackSuit
                      }`}
                    >
                      {getSuitSymbol(revealedCard.card.suit)}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Action Buttons */}
          {isMyTurn && (
            <div className={styles.actions}>
              {!drawnCard && (
                <button 
                  onClick={handleDrawCard} 
                  className={styles.button}
                  disabled={!!drawnCard || isDrawing || !isMyTurn}
                >
                  {isDrawing ? 'Drawing...' : 'Draw Card'}
                </button>
              )}
              {drawnCard && (
                <div className={styles.drawnCardArea}>
                  <div className={styles.card}>
                    <div className={styles.cardFace}>
                      <span className={styles.rank}>{drawnCard.rank}</span>
                      <span
                        className={`${styles.suit} ${
                          isRedSuit(drawnCard.suit) ? styles.redSuit : styles.blackSuit
                        }`}
                      >
                        {getSuitSymbol(drawnCard.suit)}
                      </span>
                    </div>
                  </div>
                  <p>Choose an action:</p>
                  <div className={styles.drawnCardActions}>
                    <button onClick={handleDiscardDrawnCard} className={styles.button}>
                      Discard to Pile
                    </button>
                    <p>or click one of your cards to swap</p>
                  </div>
                </div>
              )}
              {(() => {
                const topIsSpecial = gameState?.discardTop && (gameState.discardTop.rank === '7' || gameState.discardTop.rank === '8' || gameState.discardTop.rank === '9')
                const hasPendingSpecial = topIsSpecial && gameState?.pendingSpecialCard
                return (
                  <>
                    <button onClick={handleCallPablo} className={styles.button} disabled={gameState?.pabloCalled || !!drawnCard || hasPendingSpecial}>
                      Call Pablo
                    </button>
                    <button onClick={handleEndTurn} className={styles.button} disabled={!!drawnCard || hasPendingSpecial}>
                      End Turn
                    </button>
                    {drawnCard && (
                      <p className={styles.hint}>You must discard or swap the drawn card before ending your turn</p>
                    )}
                    {topIsSpecial && !drawnCard && gameState?.pendingSpecialCard && (
                      <p className={styles.hint}>Click the special card in the discard pile to use its power ({gameState.discardTop.rank})</p>
                    )}
                  </>
                )
              })()}
            </div>
          )}
        </>
      )}

      {gameState?.status === 'ended' && (
        <div className={styles.results}>
          <h2>Round Over!</h2>
          <div className={styles.scoreboard}>
            {Object.values(gameState.players)
              .sort((a, b) => a.score - b.score)
              .map((player, idx) => (
                <div key={player.id} className={styles.scoreItem}>
                  <span className={styles.rank}>{idx + 1}</span>
                  <span>{player.name}</span>
                  <span>Score: {player.score}</span>
                  {idx === 0 && <span className={styles.winner}>🏆 Winner!</span>}
                </div>
              ))}
          </div>
        </div>
      )}
    </div>
  )
}

function getSuitSymbol(suit: string): string {
  switch (suit) {
    case 'hearts':
      return '♥'
    case 'diamonds':
      return '♦'
    case 'clubs':
      return '♣'
    case 'spades':
      return '♠'
    default:
      return '?'
  }
}

