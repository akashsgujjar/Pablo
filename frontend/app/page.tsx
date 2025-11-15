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
}

export default function Home() {
  const [gameID, setGameID] = useState('')
  const [playerID, setPlayerID] = useState('')
  const [playerName, setPlayerName] = useState('')
  const [connected, setConnected] = useState(false)
  const [gameState, setGameState] = useState<GameState | null>(null)
  const [selectedCardIndex, setSelectedCardIndex] = useState<number | null>(null)
  const [drawnCard, setDrawnCard] = useState<Card | null>(null)
  const [revealedCard, setRevealedCard] = useState<{ playerID: string; index: number; card: Card } | null>(null)
  const [swapSelection, setSwapSelection] = useState<{ playerID: string; cardIndex: number } | null>(null)
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
        // Update drawn card if it exists for this player
        if (state.drawnCards && state.drawnCards[playerID]) {
          setDrawnCard(state.drawnCards[playerID])
        } else {
          setDrawnCard(null)
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
    sendMessage('drawCard', {})
  }

  const handleSwapCard = (cardIndex: number) => {
    if (drawnCard) {
      sendMessage('swapCard', { cardIndex })
      setDrawnCard(null)
      setSelectedCardIndex(null)
    }
  }

  const handleUseSpecialCard = (cardIndex: number, action: string, params: any) => {
    sendMessage('useSpecialCard', { cardIndex, action, params })
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
                          <span className={styles.suit}>{getSuitSymbol(card.suit)}</span>
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
                        } else if (isMyTurn && isSpecial) {
                          setSelectedCardIndex(idx)
                        }
                      }}
                    >
                      {card.faceUp || card.rank ? (
                        <div className={styles.cardFace}>
                          <span className={styles.rank}>{card.rank}</span>
                          <span className={styles.suit}>{getSuitSymbol(card.suit)}</span>
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

          {/* Special Card Actions */}
          {isMyTurn && selectedCardIndex !== null && myPlayer && (
            <div className={styles.specialCardMenu}>
              {myPlayer.cards[selectedCardIndex]?.rank === '7' && (
                <div>
                  <h3>Look at one of your cards:</h3>
                  {[0, 1, 2, 3].map((idx) => (
                    <button
                      key={idx}
                      onClick={() => {
                        handleUseSpecialCard(selectedCardIndex, 'look', { targetIndex: idx })
                        setSelectedCardIndex(null)
                      }}
                      className={styles.button}
                    >
                      Card {idx + 1}
                    </button>
                  ))}
                </div>
              )}
              {myPlayer.cards[selectedCardIndex]?.rank === '8' && (
                <div>
                  <h3>Look at someone else's card:</h3>
                  {otherPlayers.map((player) => (
                    <div key={player.id}>
                      <h4>{player.name}</h4>
                      {[0, 1, 2, 3].map((idx) => (
                        <button
                          key={idx}
                          onClick={() => {
                            handleUseSpecialCard(selectedCardIndex, 'spy', {
                              targetPlayerID: player.id,
                              targetIndex: idx,
                            })
                            setSelectedCardIndex(null)
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
              {myPlayer.cards[selectedCardIndex]?.rank === '9' && (
                <div>
                  <h3>Swap any two cards:</h3>
                  {!swapSelection ? (
                    <div>
                      <p>Select the first card to swap:</p>
                      <div>
                        <h4>Your cards:</h4>
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
                          <h4>{player.name}'s cards:</h4>
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
                        <h4>Your cards:</h4>
                        {[0, 1, 2, 3].map((idx) => {
                          const isSelected = swapSelection.playerID === playerID && swapSelection.cardIndex === idx
                          return (
                            <button
                              key={idx}
                              onClick={() => {
                                if (!isSelected) {
                                  handleUseSpecialCard(selectedCardIndex, 'swap', {
                                    player1ID: swapSelection.playerID,
                                    card1Index: swapSelection.cardIndex,
                                    player2ID: playerID,
                                    card2Index: idx,
                                  })
                                  setSwapSelection(null)
                                  setSelectedCardIndex(null)
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
                          <h4>{player.name}'s cards:</h4>
                          {[0, 1, 2, 3].map((idx) => {
                            const isSelected = swapSelection.playerID === player.id && swapSelection.cardIndex === idx
                            return (
                              <button
                                key={idx}
                                onClick={() => {
                                  if (!isSelected) {
                                    handleUseSpecialCard(selectedCardIndex, 'swap', {
                                      player1ID: swapSelection.playerID,
                                      card1Index: swapSelection.cardIndex,
                                      player2ID: player.id,
                                      card2Index: idx,
                                    })
                                    setSwapSelection(null)
                                    setSelectedCardIndex(null)
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
                setSelectedCardIndex(null)
                setSwapSelection(null)
              }} className={styles.button}>
                Cancel
              </button>
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
                    <span className={styles.suit}>{getSuitSymbol(revealedCard.card.suit)}</span>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Action Buttons */}
          {isMyTurn && (
            <div className={styles.actions}>
              {!drawnCard && (
                <button onClick={handleDrawCard} className={styles.button}>
                  Draw Card
                </button>
              )}
              {drawnCard && (
                <div className={styles.drawnCardArea}>
                  <div className={styles.card}>
                    <div className={styles.cardFace}>
                      <span className={styles.rank}>{drawnCard.rank}</span>
                      <span className={styles.suit}>{getSuitSymbol(drawnCard.suit)}</span>
                    </div>
                  </div>
                  <p>Click one of your cards to swap</p>
                </div>
              )}
              <button onClick={handleCallPablo} className={styles.button} disabled={gameState?.pabloCalled}>
                Call Pablo
              </button>
              <button onClick={handleEndTurn} className={styles.button}>
                End Turn
              </button>
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

