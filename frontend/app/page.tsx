'use client'

import { useState, useEffect, useRef } from 'react'
import styles from './page.module.css'

interface Card {
  suit: string
  rank: string
  faceUp: boolean
  removed?: boolean // Flag to indicate if card was removed via stacking (vs just hidden face-down)
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
  stackingEnabled: boolean
  pendingGive?: {
    actorID: string
    targetPlayerID: string
    targetIndex: number
  }
}

export default function Home() {
  const [gameID, setGameID] = useState('')
  const [playerID, setPlayerID] = useState('')
  const [playerName, setPlayerName] = useState('')
  const [connected, setConnected] = useState(false)
  const [gameState, setGameState] = useState<GameState | null>(null)
  const [drawnCard, setDrawnCard] = useState<Card | null>(null)
  const [isDrawing, setIsDrawing] = useState(false)
  const [revealedCard, setRevealedCard] = useState<{ playerID: string; index: number; card: Card } | null>(null)
  const [specialAction, setSpecialAction] = useState<
    | null
    | { type: '7' }
    | { type: '8' }
    | { type: '9'; firstSelection?: { playerID: string; cardIndex: number } }
  >(null)
  const [swapAnim, setSwapAnim] = useState<
    | null
    | {
        from: { playerID: string; index: number; rect: DOMRect; card?: Card }
        to: { playerID: string; index: number; rect: DOMRect; card?: Card }
        started: boolean
      }
  >(null)
  const [stackError, setStackError] = useState<string | null>(null)
  const [stackAttempts, setStackAttempts] = useState<{ [playerID: string]: { success: boolean; timestamp: number } }>({})
  const [isConnecting, setIsConnecting] = useState(false)
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

    setIsConnecting(true)
    const ws = new WebSocket('ws://localhost:8080/ws')
    wsRef.current = ws

    // Set a timeout for connection
    const connectionTimeout = setTimeout(() => {
      if (ws.readyState !== WebSocket.OPEN) {
        ws.close()
        setIsConnecting(false)
        alert('Connection timeout. Make sure the backend server is running on port 8080.')
      }
    }, 5000)

    ws.onopen = () => {
      clearTimeout(connectionTimeout)
      setIsConnecting(false)
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
          setSpecialAction(null)
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
      } else if (message.type === 'stackAttempt') {
        const { playerID: attemptPlayerID, success } = message.payload
        // Show popup for all players (including the player who attempted)
        setStackAttempts(prev => ({
          ...prev,
          [attemptPlayerID]: { success, timestamp: Date.now() }
        }))
        // Clear after animation duration
        setTimeout(() => {
          setStackAttempts(prev => {
            const next = { ...prev }
            delete next[attemptPlayerID]
            return next
          })
        }, 3000)
      } else if (message.type === 'stackError') {
        setStackError(message.payload.message)
        setTimeout(() => setStackError(null), 3000)
      } else if (message.type === 'swapEvent') {
        // Trigger swap animation for ALL players (including observers)
        const { player1ID, card1Index, card1, player2ID, card2Index, card2 } = message.payload
        
        // Function to find and animate cards with retry logic
        const triggerSwapAnimation = () => {
          const card1El = document.getElementById(`card-${player1ID}-${card1Index}`)
          const card2El = document.getElementById(`card-${player2ID}-${card2Index}`)
          
          if (card1El && card2El) {
            const rect1 = card1El.getBoundingClientRect()
            const rect2 = card2El.getBoundingClientRect()
            
            // Use card data from the payload (cards before swap)
            setSwapAnim({
              from: { playerID: player1ID, index: card1Index, rect: rect1, card: card1 },
              to: { playerID: player2ID, index: card2Index, rect: rect2, card: card2 },
              started: false
            })
            
            // Start animation after a brief moment
            setTimeout(() => {
              setSwapAnim(prev => {
                if (prev) {
                  return { ...prev, started: true }
                }
                return null
              })
            }, 100)
            
            // Clear animation after it completes
            setTimeout(() => {
              setSwapAnim(null)
            }, 1500)
          } else {
            // If cards not found, try again after a short delay
            // This handles cases where gameState update hasn't rendered yet
            setTimeout(() => {
              const retryCard1El = document.getElementById(`card-${player1ID}-${card1Index}`)
              const retryCard2El = document.getElementById(`card-${player2ID}-${card2Index}`)
              
              if (retryCard1El && retryCard2El) {
                const rect1 = retryCard1El.getBoundingClientRect()
                const rect2 = retryCard2El.getBoundingClientRect()
                
                setSwapAnim({
                  from: { playerID: player1ID, index: card1Index, rect: rect1, card: card1 },
                  to: { playerID: player2ID, index: card2Index, rect: rect2, card: card2 },
                  started: false
                })
                
                setTimeout(() => {
                  setSwapAnim(prev => {
                    if (prev) {
                      return { ...prev, started: true }
                    }
                    return null
                  })
                }, 100)
                
                setTimeout(() => {
                  setSwapAnim(null)
                }, 1500)
              }
            }, 200)
          }
        }
        
        // Use multiple requestAnimationFrame calls to ensure DOM is ready
        // This works for all players, including observers
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            requestAnimationFrame(() => {
              triggerSwapAnimation()
            })
          })
        })
      } else if (message.type === 'error') {
        alert(message.payload.message)
      }
    }

    ws.onerror = (error) => {
      clearTimeout(connectionTimeout)
      setIsConnecting(false)
      console.error('WebSocket error:', error)
      alert('Failed to connect to game server. Make sure the backend is running on port 8080.')
      setConnected(false)
    }

    ws.onclose = (event) => {
      clearTimeout(connectionTimeout)
      setIsConnecting(false)
      console.log('WebSocket closed:', event.code, event.reason)
      setConnected(false)
      // Only show alert if it's not a normal closure (1000) and connection was established
      if (event.code !== 1000 && event.code !== 1006) {
        // 1006 is abnormal closure (connection failed), we already show error in onerror
        alert('Connection to game server lost. Please try again.')
      }
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

  const isSwapSelected = (playerId: string, cardIndex: number) =>
    specialAction?.type === '9' &&
    specialAction.firstSelection?.playerID === playerId &&
    specialAction.firstSelection?.cardIndex === cardIndex

  const handleSpecialSwapSelection = (selection: { playerID: string; cardIndex: number }) => {
    if (specialAction?.type !== '9') return

    if (!specialAction.firstSelection) {
      setSpecialAction({ type: '9', firstSelection: selection })
      return
    }

    const first = specialAction.firstSelection
    if (first.playerID === selection.playerID && first.cardIndex === selection.cardIndex) {
      return
    }

    handleUseSpecialCardFromDiscard('9', {
      player1ID: first.playerID,
      card1Index: first.cardIndex,
      player2ID: selection.playerID,
      card2Index: selection.cardIndex,
    })
    setSpecialAction(null)
  }

  const handleSkipSpecialCard = () => {
    if (gameState?.pendingSpecialCard) {
      sendMessage('skipSpecialCard', {})
    }
    setSpecialAction(null)
  }

  const getSpecialInstruction = () => {
    if (!specialAction) return ''
    switch (specialAction.type) {
      case '7':
        return 'Click one of your cards to peek at it.'
      case '8':
        return 'Click one of your opponents’ cards to spy on it.'
      case '9':
        return specialAction.firstSelection
          ? 'Select another card to complete the swap.'
          : 'Click any card on the table to select it for swapping.'
      default:
        return ''
    }
  }

  const handleStackCard = (cardIndex: number) => {
    // Always allow attempt - backend will validate if stacking is possible
    sendMessage('stackCard', { cardIndex })
  }

  const handleMyCardClick = (idx: number) => {
    if (!isMyTurn) return
    // Pending give: choose a card to give to target
    if (gameState?.pendingGive && gameState.pendingGive.actorID === playerID) {
      const card = myPlayer?.cards[idx]
      if (!card || (!card.rank && !card.suit)) return // ignore empty placeholders
      sendMessage('giveCardToPlayer', { sourceIndex: idx })
      return
    }

    if (specialAction) {
      if (specialAction.type === '7') {
        handleUseSpecialCardFromDiscard('7', { targetIndex: idx })
        setSpecialAction(null)
        return
      }

      if (specialAction.type === '9') {
        handleSpecialSwapSelection({ playerID, cardIndex: idx })
        return
      }

      // Special action active but this card isn't valid target
      return
    }

    if (drawnCard) {
      handleSwapCard(idx)
    }
  }

  const handleMyCardDoubleClick = (idx: number) => {
    // Always allow double-click to attempt stacking - backend will validate
    handleStackCard(idx)
  }

  const handleOpponentCardClick = (targetPlayerID: string, cardIndex: number) => {
    if (!isMyTurn || !specialAction) return

    if (specialAction.type === '8') {
      handleUseSpecialCardFromDiscard('8', {
        targetPlayerID,
        targetIndex: cardIndex,
      })
      setSpecialAction(null)
      return
    }

    if (specialAction.type === '9') {
      handleSpecialSwapSelection({ playerID: targetPlayerID, cardIndex })
      return
    }

    // For other special types, clicking opponent cards does nothing
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
          <button onClick={connectWebSocket} className={styles.button} disabled={isConnecting}>
            {isConnecting ? 'Connecting...' : 'Join Game'}
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
          {/* Table: center discard pile with players around */}
          <div className={styles.table}>
            {/* Discard Pile Center */}
            {gameState.discardTop && (
              <div className={`${styles.discardPile} ${styles.discardCenter}`}>
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
                    if (
                      isMyTurn &&
                      gameState.discardTop &&
                      gameState.pendingSpecialCard === gameState.discardTop.rank
                    ) {
                      const rank = gameState.discardTop.rank
                      if (rank === '7' || rank === '8' || rank === '9') {
                        setSpecialAction({ type: rank })
                      }
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
                        <div style={{ fontSize: '10px', marginTop: '5px', color: '#ffd700', fontWeight: 'bold' }}>
                          ✨ CLICK ✨
                        </div>
                      )}
                  </div>
                </div>
              </div>
            )}

            {/* Swap animation overlay */}
            {swapAnim && (
              <div className={styles.swapOverlay}>
                {/* Connecting line between cards */}
                <svg
                  className={styles.swapLine}
                  style={{
                    position: 'fixed',
                    top: 0,
                    left: 0,
                    width: '100%',
                    height: '100%',
                    pointerEvents: 'none',
                    zIndex: 6,
                  }}
                >
                  <line
                    x1={swapAnim.started ? swapAnim.to.rect.left + swapAnim.to.rect.width / 2 : swapAnim.from.rect.left + swapAnim.from.rect.width / 2}
                    y1={swapAnim.started ? swapAnim.to.rect.top + swapAnim.to.rect.height / 2 : swapAnim.from.rect.top + swapAnim.from.rect.height / 2}
                    x2={swapAnim.started ? swapAnim.from.rect.left + swapAnim.from.rect.width / 2 : swapAnim.to.rect.left + swapAnim.to.rect.width / 2}
                    y2={swapAnim.started ? swapAnim.from.rect.top + swapAnim.from.rect.height / 2 : swapAnim.to.rect.top + swapAnim.to.rect.height / 2}
                    stroke="#ffd700"
                    strokeWidth="4"
                    strokeDasharray="10,5"
                    opacity={swapAnim.started ? 0.8 : 0.4}
                    style={{
                      filter: 'drop-shadow(0 0 8px rgba(255, 215, 0, 0.8))',
                      transition: 'all 0.6s ease',
                    }}
                  />
                </svg>
                
                {/* First card (moving to second position) */}
                <div
                  className={`${styles.card} ${styles.swapAnimCard} ${styles.swapCardGlow}`}
                  style={{
                    position: 'fixed',
                    top: swapAnim.from.rect.top,
                    left: swapAnim.from.rect.left,
                    width: swapAnim.from.rect.width,
                    height: swapAnim.from.rect.height,
                    transform: swapAnim.started
                      ? `translate(${swapAnim.to.rect.left - swapAnim.from.rect.left}px, ${swapAnim.to.rect.top - swapAnim.from.rect.top}px) scale(1.1)`
                      : 'translate(0, 0) scale(1)',
                    zIndex: 10,
                  }}
                >
                  {swapAnim.from.card && (swapAnim.from.card.faceUp || swapAnim.from.card.rank) ? (
                    <div className={styles.cardFace}>
                      <span className={styles.rank}>{swapAnim.from.card.rank}</span>
                      <span
                        className={`${styles.suit} ${
                          isRedSuit(swapAnim.from.card.suit) ? styles.redSuit : styles.blackSuit
                        }`}
                      >
                        {getSuitSymbol(swapAnim.from.card.suit)}
                      </span>
                    </div>
                  ) : (
                    <div className={styles.cardBackPattern}>🎴</div>
                  )}
                </div>
                
                {/* Second card (moving to first position) */}
                <div
                  className={`${styles.card} ${styles.swapAnimCard} ${styles.swapCardGlow}`}
                  style={{
                    position: 'fixed',
                    top: swapAnim.to.rect.top,
                    left: swapAnim.to.rect.left,
                    width: swapAnim.to.rect.width,
                    height: swapAnim.to.rect.height,
                    transform: swapAnim.started
                      ? `translate(${swapAnim.from.rect.left - swapAnim.to.rect.left}px, ${swapAnim.from.rect.top - swapAnim.to.rect.top}px) scale(1.1)`
                      : 'translate(0, 0) scale(1)',
                    zIndex: 10,
                  }}
                >
                  {swapAnim.to.card && (swapAnim.to.card.faceUp || swapAnim.to.card.rank) ? (
                    <div className={styles.cardFace}>
                      <span className={styles.rank}>{swapAnim.to.card.rank}</span>
                      <span
                        className={`${styles.suit} ${
                          isRedSuit(swapAnim.to.card.suit) ? styles.redSuit : styles.blackSuit
                        }`}
                      >
                        {getSuitSymbol(swapAnim.to.card.suit)}
                      </span>
                    </div>
                  ) : (
                    <div className={styles.cardBackPattern}>🎴</div>
                  )}
                </div>
              </div>
            )}
            {/* Pending Give Instruction */}
            {gameState?.pendingGive && gameState.pendingGive.actorID === playerID && (
              <div className={styles.specialInstruction}>
                <p>
                  Choose one of your cards to give to{' '}
                  {gameState.players[gameState.pendingGive.targetPlayerID]?.name || 'the player'}.
                </p>
                <p className={styles.secondaryHint}>
                  It will go into their empty slot automatically.
                </p>
              </div>
            )}
            {isMyTurn &&
              specialAction &&
              gameState?.pendingSpecialCard === gameState?.discardTop?.rank && (
                <div className={styles.specialInstruction}>
                  <p>{getSpecialInstruction()}</p>
                  {specialAction.type === '9' && specialAction.firstSelection && (
                    <p className={styles.secondaryHint}>
                      First card selected. Choose another card to complete the swap.
                    </p>
                  )}
                  <button onClick={handleSkipSpecialCard} className={styles.button}>
                    Skip Special Card
                  </button>
                </div>
              )}
            
            {/* Players ring around the discard pile (other players) */}
            <div className={styles.playersRing}>
              {(() => {
                const seatedPlayers = otherPlayers
                const count = seatedPlayers.length
                const radius = 220
                return seatedPlayers.map((player, i) => {
                  const angle = (2 * Math.PI * i) / (count || 1)
                  const x = Math.cos(angle) * radius
                  const y = Math.sin(angle) * radius
                  return (
                    <div
                      key={player.id}
                      className={styles.playerSeat}
                      style={{
                        top: `calc(50% + ${y}px)`,
                        left: `calc(50% + ${x}px)`,
                      }}
                    >
                      <div className={styles.playerArea}>
                        <h3>
                          {player.name} {gameState.currentPlayer === player.id && '👈'}
                        </h3>
                        <div className={styles.myCardsContainer}>
                          <div className={styles.opponentGrid}>
                            {Array.from({ length: 4 }, (_, idx) => {
                              const card = player.cards[idx] || { rank: '', suit: '', faceUp: false, removed: false }
                              // Calculate explicit grid position: row = Math.floor(idx / 2) + 1, col = (idx % 2) + 1
                              const gridRow = Math.floor(idx / 2) + 1
                              const gridCol = (idx % 2) + 1
                              
                              // For the first 4 cards: if card was removed via stacking, render as invisible placeholder
                              if (card.removed) {
                                return (
                                  <div
                                    key={idx}
                                    id={`card-${player.id}-${idx}`}
                                    className={styles.card}
                                    style={{
                                      visibility: 'hidden',
                                      pointerEvents: 'none',
                                      gridRow: gridRow,
                                      gridColumn: gridCol,
                                    }}
                                  >
                                    <div className={styles.cardBackPattern}>🎴</div>
                                  </div>
                                )
                              }
                              
                              // Render actual cards (face-down or face-up)
                              // - If rank/suit are empty but not removed, it's a face-down card (show card back)
                              // - If rank/suit exist and faceUp, show card face
                              // - If rank/suit exist but not faceUp, show card back
                              const hasStackAttempt = stackAttempts[player.id] !== undefined
                              return (
                                <div
                                  key={idx}
                                  id={`card-${player.id}-${idx}`}
                                  className={`${styles.card} ${styles.cardBack} ${
                                    isSwapSelected(player.id, idx) ? styles.swapSelected : ''
                                  } ${hasStackAttempt ? styles.stackAttemptCard : ''}`}
                                  onClick={() => handleOpponentCardClick(player.id, idx)}
                                  onDoubleClick={() => {
                                    if (gameState?.stackingEnabled && !specialAction && !drawnCard && !card.faceUp && !card.removed) {
                                      sendMessage('stackOpponentCard', { targetPlayerID: player.id, cardIndex: idx })
                                    }
                                  }}
                                  style={{
                                    cursor:
                                      specialAction && (specialAction.type === '8' || specialAction.type === '9')
                                        ? 'pointer'
                                        : gameState?.stackingEnabled && !specialAction && !drawnCard && !card.faceUp && !card.removed
                                          ? 'pointer'
                                          : 'default',
                                    gridRow: gridRow,
                                    gridColumn: gridCol,
                                  }}
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
                              )
                            })}
                          </div>
                          {(() => {
                            // Only show penalty section if there are actual penalty cards (non-removed cards at index 4+)
                            // For opponents, face-down cards have empty rank/suit but removed: false, so we check !removed
                            const penaltyCards = player.cards.slice(4).filter(card => !card.removed)
                            return penaltyCards.length > 0 && (
                              <div className={styles.penaltyCardsContainer}>
                                <div className={styles.penaltyLabel}>Penalty Cards</div>
                                <div className={styles.penaltyCards}>
                                  {player.cards.slice(4).map((card, idx) => {
                                    const actualIdx = idx + 4
                                    // Skip removed cards in penalty section
                                    // Face-down cards (empty rank/suit but not removed) should still be shown as card backs
                                    if (card.removed) {
                                      return null
                                    }
                                    const hasStackAttempt = stackAttempts[player.id] !== undefined
                                    return (
                                      <div
                                        key={actualIdx}
                                        id={`card-${player.id}-${actualIdx}`}
                                        className={`${styles.card} ${styles.penaltyCard} ${styles.cardBack} ${
                                          isSwapSelected(player.id, actualIdx) ? styles.swapSelected : ''
                                        } ${hasStackAttempt ? styles.stackAttemptCard : ''}`}
                                        onClick={() => handleOpponentCardClick(player.id, actualIdx)}
                                        onDoubleClick={() => {
                                          if (gameState?.stackingEnabled && !specialAction && !drawnCard && !card.faceUp && !card.removed) {
                                            sendMessage('stackOpponentCard', { targetPlayerID: player.id, cardIndex: actualIdx })
                                          }
                                        }}
                                        style={{
                                          cursor:
                                            specialAction && (specialAction.type === '8' || specialAction.type === '9')
                                              ? 'pointer'
                                              : gameState?.stackingEnabled && !specialAction && !drawnCard && !card.faceUp && !card.removed
                                                ? 'pointer'
                                                : 'default',
                                        }}
                                      >
                                        <div className={styles.cardBackPattern}>🎴</div>
                                      </div>
                                    )
                                  })}
                                </div>
                              </div>
                            )
                          })()}
                        </div>
                      </div>
                    </div>
                  )
                })
              })()}
            </div>
          </div>

          {/* My Cards */}
          {myPlayer && (
            <div className={styles.myArea}>
              <h2>Your Cards {isMyTurn && '👈 Your Turn'}</h2>
              <div className={styles.myCardsContainer}>
                <div className={styles.myGrid}>
                  {Array.from({ length: 4 }, (_, idx) => {
                    const card = myPlayer.cards[idx] || { rank: '', suit: '', faceUp: false, removed: false }
                    // Calculate explicit grid position: row = Math.floor(idx / 2) + 1, col = (idx % 2) + 1
                    const gridRow = Math.floor(idx / 2) + 1
                    const gridCol = (idx % 2) + 1
                    
                    // Render removed/empty cards as invisible placeholders to maintain positions
                    if (card.removed || (!card.rank && !card.suit)) {
                      return (
                        <div
                          key={`card-slot-${idx}`}
                          id={`card-${playerID}-${idx}`}
                          className={styles.card}
                          style={{
                            visibility: 'hidden',
                            pointerEvents: 'none',
                            gridRow: gridRow,
                            gridColumn: gridCol,
                          }}
                        >
                          <div className={styles.cardBackPattern}>🎴</div>
                        </div>
                      )
                    }
                    const isSpecial = card.rank === '7' || card.rank === '8' || card.rank === '9'
                    return (
                      <div
                        key={`card-slot-${idx}`}
                        id={`card-${playerID}-${idx}`}
                        className={`${styles.card} ${card.faceUp ? styles.cardFace : styles.cardBack} ${
                          isSwapSelected(playerID, idx) ? styles.swapSelected : ''
                      } ${gameState?.stackingEnabled ? styles.stackableCard : ''}`}
                      onClick={() => handleMyCardClick(idx)}
                      onDoubleClick={() => handleMyCardDoubleClick(idx)}
                      title="Double-click to attempt stacking"
                      style={{
                        cursor:
                          gameState?.pendingGive && gameState.pendingGive.actorID === playerID
                            ? 'pointer'
                            : 'pointer',
                        gridRow: gridRow,
                        gridColumn: gridCol,
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
                {(() => {
                  // Only show penalty section if there are actual penalty cards (non-empty cards at index 4+)
                  const penaltyCards = myPlayer.cards.slice(4).filter(card => card.rank || card.suit)
                  return penaltyCards.length > 0 && (
                    <div className={styles.penaltyCardsContainer}>
                      <div className={styles.penaltyLabel}>Penalty Cards</div>
                      <div className={styles.penaltyCards}>
                        {myPlayer.cards.slice(4).map((card, idx) => {
                          const actualIdx = idx + 4
                          // Skip empty cards in penalty section - they shouldn't appear here
                          if (!card.rank && !card.suit) {
                            return null
                          }
                        const isSpecial = card.rank === '7' || card.rank === '8' || card.rank === '9'
                        return (
                          <div
                            key={actualIdx}
                            id={`card-${playerID}-${actualIdx}`}
                            className={`${styles.card} ${styles.penaltyCard} ${card.faceUp ? styles.cardFace : styles.cardBack} ${
                              isSwapSelected(playerID, actualIdx) ? styles.swapSelected : ''
                            } ${gameState?.stackingEnabled && !specialAction && !drawnCard ? styles.stackableCard : ''}`}
                            onClick={() => handleMyCardClick(actualIdx)}
                            onDoubleClick={() => handleMyCardDoubleClick(actualIdx)}
                            title="Double-click to attempt stacking"
                            style={{
                              cursor:
                                specialAction
                                  ? specialAction.type === '7' || specialAction.type === '9'
                                    ? 'pointer'
                                    : 'default'
                                  : drawnCard
                                    ? 'pointer'
                                    : gameState?.stackingEnabled && !specialAction && !drawnCard
                                      ? 'pointer'
                                      : 'default',
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
                  </div>
                  )
                })()}
              </div>
              <div className={styles.myTotal}>
                Total: {getPlayerTotal(myPlayer)}
              </div>
            </div>
          )}


          {/* Stack Error Display */}
          {stackError && (
            <div className={styles.stackError}>
              <div className={styles.stackErrorContent}>
                <h3>❌ Stack Failed!</h3>
                <p>{stackError}</p>
              </div>
            </div>
          )}

          {/* Stack Attempt Popups for All Players */}
          {Object.entries(stackAttempts).map(([attemptPlayerID, attempt]) => {
            const attemptPlayer = gameState?.players[attemptPlayerID]
            if (!attemptPlayer) return null
            return (
              <div
                key={attemptPlayerID}
                className={`${styles.stackAttemptAnimation} ${
                  attempt.success ? styles.stackSuccess : styles.stackFailure
                }`}
              >
                <div className={styles.stackAttemptContent}>
                  <h3>{attemptPlayer.name}</h3>
                  <p>
                    {attempt.success ? '✅ Successfully stacked!' : '❌ Failed to stack - penalty card added'}
                  </p>
                </div>
              </div>
            )
          })}

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
                const topIsSpecial = Boolean(
                  gameState?.discardTop &&
                    (gameState.discardTop.rank === '7' ||
                      gameState.discardTop.rank === '8' ||
                      gameState.discardTop.rank === '9')
                )
                const hasPendingSpecial = Boolean(topIsSpecial && gameState?.pendingSpecialCard)
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

