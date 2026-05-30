package network

import (
	"encoding/json"
	"poker-duel/internal/game"
	"time"

	"github.com/gorilla/websocket"
)

type PlayerState struct {
	Conn   *websocket.Conn
	Name   string
	Cards  []game.Card
	Chips  int
	Bet    int
	Folded bool
	IsTurn bool
	AllIn  bool
}

type GameRoom struct {
	ID               string
	Players          []*PlayerState
	Deck             []game.Card
	Table            []game.Card
	Pot              int
	CurrentTurn      int
	GamePhase        string // "waiting", "preflop", "flop", "turn", "river", "showdown", "finished"
	SmallBlind       int
	BigBlind         int
	CurrentBet       int
	LastAction       string // "check", "call", "bet", "raise", "fold", ""
	DealerPos        int
	Timer            *time.Timer
	TimerSeconds     int
	Hub              *Hub // Reference to hub so timeout can send action
	PlayersActed     int  // Number of players who have acted in current betting round

}

func (room *GameRoom) Broadcast(message interface{}) {
	for _, p := range room.Players {
		if p.Conn != nil {
			room.SendToPlayer(p, message)
		}
	}
}

func (room *GameRoom) SendToPlayer(player *PlayerState, baseMessage interface{}) {
	playerInfos := make([]map[string]interface{}, len(room.Players))
	for i, p := range room.Players {
		cards := []map[string]interface{}{}
		if room.GamePhase == "showdown" || room.GamePhase == "finished" || p.Conn == player.Conn {
			for _, c := range p.Cards {
				cards = append(cards, map[string]interface{}{"suit": c.Suit, "value": c.Value})
			}
		}
		playerInfos[i] = map[string]interface{}{
			"name":    p.Name,
			"chips":   p.Chips,
			"bet":     p.Bet,
			"folded":  p.Folded,
			"is_turn": p.IsTurn,
			"all_in":  p.AllIn,
			"cards":   cards,
		}
	}

	tableCards := []map[string]interface{}{}
	for _, c := range room.Table {
		tableCards = append(tableCards, map[string]interface{}{"suit": c.Suit, "value": c.Value})
	}

	fullMsg := make(map[string]interface{})
	if baseMap, ok := baseMessage.(map[string]interface{}); ok {
		for k, v := range baseMap {
			fullMsg[k] = v
		}
	}
	fullMsg["type"] = "game_state"
	fullMsg["room"] = room.ID
	fullMsg["phase"] = room.GamePhase
	fullMsg["pot"] = room.Pot
	fullMsg["table_cards"] = tableCards
	fullMsg["players"] = playerInfos
	fullMsg["current_turn"] = room.CurrentTurn
	fullMsg["current_bet"] = room.CurrentBet
	fullMsg["last_bet"] = room.CurrentBet
	fullMsg["last_action"] = room.LastAction
	fullMsg["dealer_pos"] = room.DealerPos

	msg, _ := json.Marshal(fullMsg)
	player.Conn.WriteMessage(websocket.TextMessage, msg)
}

func (room *GameRoom) GetGameState() map[string]interface{} {
	return map[string]interface{}{}
}

func (room *GameRoom) markZeroChipPlayersAllIn() {
	for _, p := range room.Players {
		if !p.Folded && p.Chips <= 0 {
			p.AllIn = true
		}
	}
}

func (room *GameRoom) resetBettingRound() {
	for _, p := range room.Players {
		p.Bet = 0
		p.IsTurn = false
	}
	room.CurrentBet = 0
	room.LastAction = ""
	room.PlayersActed = 0
	for _, p := range room.Players {
		if !p.Folded && p.AllIn {
			room.PlayersActed++
		}
	}
}

func (room *GameRoom) shouldResolveRound() bool {
	activeCount := 0
	allMatched := true
	allInCount := 0

	for _, p := range room.Players {
		if p.Folded {
			continue
		}
		activeCount++
		if !p.AllIn && p.Bet != room.CurrentBet {
			allMatched = false
		}
		if p.AllIn {
			allInCount++
		}
	}

	if activeCount <= 1 {
		return false
	}

	if !allMatched || room.PlayersActed < 2 {
		return false
	}

	if allInCount >= 1 {
		room.dealRemainingCards()
		room.GamePhase = "showdown"
		room.DetermineWinner()
		return true
	}

	room.NextPhase()
	return true
}

func (room *GameRoom) StartGame() {
	room.Deck = game.NewDeck()
	room.Table = []game.Card{}
	room.Pot = 0
	room.GamePhase = "preflop"
	room.CurrentBet = room.BigBlind
	room.LastAction = ""
	room.TimerSeconds = 20
	room.PlayersActed = 0

	for i, p := range room.Players {
		p.Cards = room.Deck[i*2 : (i+1)*2]
		p.Bet = 0
		p.Folded = false
		p.AllIn = false
		p.IsTurn = false
	}

	// Heads-Up: Dealer is SB, other is BB
	sbPos := room.DealerPos
	bbPos := (room.DealerPos + 1) % 2

	room.Players[sbPos].Bet = room.SmallBlind
	room.Players[sbPos].Chips -= room.SmallBlind
	room.Players[bbPos].Bet = room.BigBlind
	room.Players[bbPos].Chips -= room.BigBlind
	room.Pot = room.SmallBlind + room.BigBlind

	room.markZeroChipPlayersAllIn()

	// Preflop: first to act is SB (dealer). BB has already posted a blind and is treated as having acted.
	if room.Players[sbPos].AllIn {
		room.CurrentTurn = bbPos
	} else {
		room.CurrentTurn = sbPos
	}
	room.Players[room.CurrentTurn].IsTurn = true
	room.PlayersActed = 0
	for _, p := range room.Players {
		if !p.Folded && p.AllIn {
			room.PlayersActed++
		}
	}

	if room.shouldResolveRound() {
		return
	}

	room.StartTimer()
	room.Broadcast(room.GetGameState())
}

func (room *GameRoom) NextPhase() {
	room.StopTimer()

	switch room.GamePhase {
	case "preflop":
		room.GamePhase = "flop"
		room.Table = append(room.Table, room.Deck[4:7]...)
	case "flop":
		room.GamePhase = "turn"
		room.Table = append(room.Table, room.Deck[7])
	case "turn":
		room.GamePhase = "river"
		room.Table = append(room.Table, room.Deck[8])
	case "river":
		room.GamePhase = "showdown"
		room.DetermineWinner()
		return
	}

	room.markZeroChipPlayersAllIn()
	// Reset bets for new phase and count players who are already all-in.
	room.resetBettingRound()

	// Postflop: first to act is BB
	bbPos := (room.DealerPos + 1) % 2
	room.CurrentTurn = bbPos
	room.Players[room.CurrentTurn].IsTurn = true

	if room.shouldResolveRound() {
		return
	}

	room.StartTimer()
	room.Broadcast(room.GetGameState())
}

func (room *GameRoom) dealRemainingCards() {
	if room.GamePhase == "preflop" && len(room.Table) == 0 {
		room.Table = append(room.Table, room.Deck[4:7]...)
	}
	if len(room.Table) < 4 {
		room.Table = append(room.Table, room.Deck[7])
	}
	if len(room.Table) < 5 {
		room.Table = append(room.Table, room.Deck[8])
	}
}

func (room *GameRoom) clearPlayerTurns() {
	for _, p := range room.Players {
		p.IsTurn = false
	}
	room.CurrentTurn = -1
}

func (room *GameRoom) PlayerFold(playerIndex int) {
	room.StopTimer()
	room.Players[playerIndex].Folded = true
	room.Players[playerIndex].IsTurn = false
	room.LastAction = "fold"

	// Check if only one player remains
	activePlayers := 0
	winnerIndex := 0
	for i, p := range room.Players {
		if !p.Folded {
			activePlayers++
			winnerIndex = i
		}
	}

	if activePlayers == 1 {
		room.clearPlayerTurns()
		room.Players[winnerIndex].Chips += room.Pot
		room.GamePhase = "finished"
		room.SwitchDealer()
		room.Broadcast(room.GetGameState())

		// Check if any player has 0 or less chips
		gameOver := false
		var loserIndex int
		for i, p := range room.Players {
			if p.Chips <= 0 {
				gameOver = true
				loserIndex = i
				break
			}
		}

		if gameOver {
			// Send game_over message to both players
			for i, p := range room.Players {
				if p.Conn != nil {
					won := i != loserIndex
					msg, _ := json.Marshal(map[string]interface{}{
						"type": "game_over",
						"won":  won,
					})
					p.Conn.WriteMessage(websocket.TextMessage, msg)
				}
			}
			return
		}

		// Start new round after a short delay
		time.AfterFunc(2*time.Second, func() {
			room.StartGame()
		})
		return
	}

	room.NextTurn()
}

func (room *GameRoom) PlayerCheck(playerIndex int) {
	room.StopTimer()
	room.Players[playerIndex].IsTurn = false
	room.PlayersActed++
	room.LastAction = "check"
	room.NextTurn()
}

func (room *GameRoom) PlayerCall(playerIndex int) {
	room.StopTimer()
	callAmount := room.CurrentBet - room.Players[playerIndex].Bet
	if callAmount >= room.Players[playerIndex].Chips {
		callAmount = room.Players[playerIndex].Chips
		room.Players[playerIndex].AllIn = true
	}

	room.Players[playerIndex].Chips -= callAmount
	room.Players[playerIndex].Bet += callAmount
	room.Pot += callAmount
	if room.Players[playerIndex].Chips == 0 {
		room.Players[playerIndex].AllIn = true
	}
	room.Players[playerIndex].IsTurn = false
	room.PlayersActed++
	room.LastAction = "call"

	room.NextTurn()
}

func (room *GameRoom) PlayerBet(playerIndex int, amount int) {
	room.StopTimer()
	totalBet := amount
	betAmount := totalBet - room.Players[playerIndex].Bet

	if betAmount >= room.Players[playerIndex].Chips {
		betAmount = room.Players[playerIndex].Chips
		room.Players[playerIndex].AllIn = true
		totalBet = room.Players[playerIndex].Bet + betAmount
	}

	room.Players[playerIndex].Chips -= betAmount
	room.Players[playerIndex].Bet = totalBet
	room.Pot += betAmount
	if room.Players[playerIndex].Chips == 0 {
		room.Players[playerIndex].AllIn = true
	}
	room.CurrentBet = totalBet
	room.Players[playerIndex].IsTurn = false
	room.PlayersActed = 1
	room.LastAction = "bet"

	room.NextTurn()
}

func (room *GameRoom) PlayerRaise(playerIndex int, amount int) {
	room.StopTimer()
	// The amount is the target total bet after raising, not the additional increment.
	totalBet := amount
	raiseAmount := totalBet - room.Players[playerIndex].Bet

	if raiseAmount >= room.Players[playerIndex].Chips {
		raiseAmount = room.Players[playerIndex].Chips
		room.Players[playerIndex].AllIn = true
		totalBet = room.Players[playerIndex].Bet + raiseAmount
	}

	if raiseAmount < 0 {
		raiseAmount = 0
		totalBet = room.Players[playerIndex].Bet
	}

	room.Players[playerIndex].Chips -= raiseAmount
	room.Players[playerIndex].Bet = totalBet
	room.Pot += raiseAmount
	if room.Players[playerIndex].Chips == 0 {
		room.Players[playerIndex].AllIn = true
	}
	room.CurrentBet = totalBet
	room.Players[playerIndex].IsTurn = false
	room.PlayersActed = 1
	room.LastAction = "raise"

	room.NextTurn()
}

func (room *GameRoom) NextTurn() {
	room.markZeroChipPlayersAllIn()

	if room.shouldResolveRound() {
		return
	}

	// Find next player (skip folded or all-in players)
	room.CurrentTurn = (room.CurrentTurn + 1) % 2
	for room.Players[room.CurrentTurn].Folded || room.Players[room.CurrentTurn].AllIn {
		room.CurrentTurn = (room.CurrentTurn + 1) % 2
		// If we looped back and everyone is folded/all-in, check if we need to deal remaining cards
		if room.Players[room.CurrentTurn].Folded || room.Players[room.CurrentTurn].AllIn {
			// Check if there are active players left
			anyActive := false
			for _, p := range room.Players {
				if !p.Folded && !p.AllIn {
					anyActive = true
					break
				}
			}
			if !anyActive {
				// Deal remaining cards and go to showdown
				room.dealRemainingCards()
				room.GamePhase = "showdown"
				room.DetermineWinner()
				return
			}
		}
	}
	room.Players[room.CurrentTurn].IsTurn = true

	room.StartTimer()
	room.Broadcast(room.GetGameState())
}

func (room *GameRoom) DetermineWinner() {
	room.StopTimer()
	room.clearPlayerTurns()
	var winners []int
	bestRank := -1
	var bestKickers []int

	for i, p := range room.Players {
		if p.Folded {
			continue
		}

		rank, kickers := game.EvaluateHand(p.Cards, room.Table)
		if rank > bestRank {
			bestRank = rank
			bestKickers = kickers
			winners = []int{i}
		} else if rank == bestRank {
			better := false
			equal := true
			for j := 0; j < len(kickers) && j < len(bestKickers); j++ {
				if kickers[j] > bestKickers[j] {
					better = true
					equal = false
					break
				} else if kickers[j] < bestKickers[j] {
					equal = false
					break
				}
			}
			if better {
				winners = []int{i}
				bestKickers = kickers
			} else if equal {
				winners = append(winners, i)
			}
		}
	}

	if len(room.Players) == 2 && !room.Players[0].Folded && !room.Players[1].Folded {
		p0Bet := room.Players[0].Bet
		p1Bet := room.Players[1].Bet
		minBet := p0Bet
		if p1Bet < minBet {
			minBet = p1Bet
		}
		mainPot := minBet * 2
		sidePot := room.Pot - mainPot

		if len(winners) == 1 {
			winner := winners[0]
			loser := 1 - winner
			if room.Players[winner].Bet == minBet && room.Players[loser].Bet > minBet {
				room.Players[winner].Chips += mainPot
				room.Players[loser].Chips += sidePot
			} else {
				room.Players[winner].Chips += room.Pot
			}
		} else {
			room.Players[0].Chips += mainPot / 2
			room.Players[1].Chips += mainPot / 2
			if sidePot > 0 {
				if p0Bet > p1Bet {
					room.Players[0].Chips += sidePot
				} else if p1Bet > p0Bet {
					room.Players[1].Chips += sidePot
				} else {
					room.Players[0].Chips += sidePot / 2
					room.Players[1].Chips += sidePot / 2
				}
			}
		}
	} else {
		prizePerWinner := room.Pot / len(winners)
		for _, idx := range winners {
			room.Players[idx].Chips += prizePerWinner
		}
	}

	room.SwitchDealer()
	room.GamePhase = "finished"
	room.Broadcast(room.GetGameState())

	// Check if any player has 0 or less chips
	gameOver := false
	var loserIndex int
	for i, p := range room.Players {
		if p.Chips <= 0 {
			gameOver = true
			loserIndex = i
			break
		}
	}

	if gameOver {
		// Send game_over message to both players
		for i, p := range room.Players {
			if p.Conn != nil {
				won := i != loserIndex
				msg, _ := json.Marshal(map[string]interface{}{
					"type": "game_over",
					"won":  won,
				})
				p.Conn.WriteMessage(websocket.TextMessage, msg)
			}
		}
		return
	}

	// Start new round after a short delay
	time.AfterFunc(2*time.Second, func() {
		room.StartGame()
	})
}

func (room *GameRoom) SwitchDealer() {
	room.DealerPos = (room.DealerPos + 1) % 2
}

func (room *GameRoom) StartTimer() {
	room.Timer = time.AfterFunc(time.Duration(room.TimerSeconds)*time.Second, func() {
		// When timeout happens, we need to hold the hub mutex
		// So we'll signal the hub to process the timeout action
		if room.Hub != nil {
			room.Hub.processTimeout(room.ID)
		}
	})
}

func (room *GameRoom) StopTimer() {
	if room.Timer != nil {
		room.Timer.Stop()
		room.Timer = nil
	}
}

func (room *GameRoom) HandleTimeout() {
	player := room.Players[room.CurrentTurn]
	if player.Folded || player.AllIn {
		return
	}

	// Check if player can check
	if player.Bet == room.CurrentBet {
		room.PlayerCheck(room.CurrentTurn)
	} else {
		room.PlayerFold(room.CurrentTurn)
	}
}
