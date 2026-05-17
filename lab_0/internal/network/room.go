package network

import (
	"encoding/json"
	"poker-duel/internal/game"

	"github.com/gorilla/websocket"
)

type PlayerState struct {
	Conn    *websocket.Conn
	Name    string
	Cards   []game.Card
	Chips   int
	Bet     int
	Folded  bool
	IsTurn  bool
	AllIn   bool
}

type GameRoom struct {
	ID           string
	Players      []*PlayerState
	Deck         []game.Card
	Table        []game.Card
	Pot          int
	CurrentTurn  int
	GamePhase    string // "waiting", "preflop", "flop", "turn", "river", "showdown", "finished"
	SmallBlind   int
	BigBlind     int
	MinBet       int
	LastBet      int
	DealerPos    int
}

func (room *GameRoom) Broadcast(message interface{}) {
	msg, _ := json.Marshal(message)
	for _, p := range room.Players {
		if p.Conn != nil {
			p.Conn.WriteMessage(websocket.TextMessage, msg)
		}
	}
}

func (room *GameRoom) GetGameState() map[string]interface{} {
	playerInfos := make([]map[string]interface{}, len(room.Players))
	for i, p := range room.Players {
		cards := []map[string]interface{}{}
		if room.GamePhase == "showdown" || room.GamePhase == "finished" {
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

	return map[string]interface{}{
		"type":         "game_state",
		"room":         room.ID,
		"phase":        room.GamePhase,
		"pot":          room.Pot,
		"table_cards":  tableCards,
		"players":      playerInfos,
		"current_turn": room.CurrentTurn,
		"min_bet":      room.MinBet,
		"last_bet":     room.LastBet,
	}
}

func (room *GameRoom) StartGame() {
	room.Deck = game.NewDeck()
	room.Table = []game.Card{}
	room.Pot = 0
	room.GamePhase = "preflop"
	room.LastBet = room.BigBlind
	room.MinBet = room.BigBlind

	for i, p := range room.Players {
		p.Cards = room.Deck[i*2 : (i+1)*2]
		p.Bet = 0
		p.Folded = false
		p.AllIn = false
		p.IsTurn = false
	}

	// Постинг блайндов
	sbPos := (room.DealerPos + 1) % len(room.Players)
	bbPos := (room.DealerPos + 2) % len(room.Players)

	room.Players[sbPos].Bet = room.SmallBlind
	room.Players[sbPos].Chips -= room.SmallBlind
	room.Players[bbPos].Bet = room.BigBlind
	room.Players[bbPos].Chips -= room.BigBlind
	room.Pot = room.SmallBlind + room.BigBlind

	// Первый ход после блайндов
	room.CurrentTurn = (room.DealerPos + 3) % len(room.Players)
	room.Players[room.CurrentTurn].IsTurn = true

	room.Broadcast(room.GetGameState())
}

func (room *GameRoom) NextPhase() {
	switch room.GamePhase {
	case "preflop":
		room.GamePhase = "flop"
		room.Table = append(room.Table, room.Deck[4:7]...) // Flop: 3 карты
	case "flop":
		room.GamePhase = "turn"
		room.Table = append(room.Table, room.Deck[7]) // Turn: 1 карта
	case "turn":
		room.GamePhase = "river"
		room.Table = append(room.Table, room.Deck[8]) // River: 1 карта
	case "river":
		room.GamePhase = "showdown"
		room.DetermineWinner()
		return
	}

	// Сброс ставок для новой фазы
	for _, p := range room.Players {
		p.Bet = 0
		p.IsTurn = false
	}
	room.LastBet = 0
	room.MinBet = room.BigBlind
	room.CurrentTurn = (room.DealerPos + 1) % len(room.Players)
	
	// Найти первого не фолднувшего игрока
	for room.Players[room.CurrentTurn].Folded {
		room.CurrentTurn = (room.CurrentTurn + 1) % len(room.Players)
	}
	room.Players[room.CurrentTurn].IsTurn = true

	room.Broadcast(room.GetGameState())
}

func (room *GameRoom) PlayerFold(playerIndex int) {
	room.Players[playerIndex].Folded = true
	room.Players[playerIndex].IsTurn = false

	// Проверка, остался ли только один игрок
	activePlayers := 0
	winnerIndex := 0
	for i, p := range room.Players {
		if !p.Folded {
			activePlayers++
			winnerIndex = i
		}
	}

	if activePlayers == 1 {
		room.Players[winnerIndex].Chips += room.Pot
		room.GamePhase = "finished"
		room.Broadcast(room.GetGameState())
		return
	}

	room.NextTurn()
}

func (room *GameRoom) PlayerCall(playerIndex int) {
	callAmount := room.LastBet - room.Players[playerIndex].Bet
	if callAmount > room.Players[playerIndex].Chips {
		callAmount = room.Players[playerIndex].Chips
		room.Players[playerIndex].AllIn = true
	}

	room.Players[playerIndex].Chips -= callAmount
	room.Players[playerIndex].Bet += callAmount
	room.Pot += callAmount
	room.Players[playerIndex].IsTurn = false

	room.NextTurn()
}

func (room *GameRoom) PlayerRaise(playerIndex int, amount int) {
	if amount < room.MinBet {
		amount = room.MinBet
	}
	totalBet := room.LastBet + amount
	raiseAmount := totalBet - room.Players[playerIndex].Bet

	if raiseAmount > room.Players[playerIndex].Chips {
		raiseAmount = room.Players[playerIndex].Chips
		room.Players[playerIndex].AllIn = true
		totalBet = room.Players[playerIndex].Bet + raiseAmount
	}

	room.Players[playerIndex].Chips -= raiseAmount
	room.Players[playerIndex].Bet = totalBet
	room.Pot += raiseAmount
	room.LastBet = totalBet
	room.MinBet = amount
	room.Players[playerIndex].IsTurn = false

	room.NextTurn()
}

func (room *GameRoom) NextTurn() {
	// Проверить, все ли игроки сделали равные ставки или фолднули
	allMatched := true
	activeCount := 0
	for _, p := range room.Players {
		if !p.Folded {
			activeCount++
			if p.Bet != room.LastBet && !p.AllIn {
				allMatched = false
			}
		}
	}

	if allMatched && activeCount > 1 {
		room.NextPhase()
		return
	}

	// Найти следующего игрока
	room.CurrentTurn = (room.CurrentTurn + 1) % len(room.Players)
	for room.Players[room.CurrentTurn].Folded || room.Players[room.CurrentTurn].AllIn {
		room.CurrentTurn = (room.CurrentTurn + 1) % len(room.Players)
		if room.CurrentTurn == room.DealerPos {
			break
		}
	}
	room.Players[room.CurrentTurn].IsTurn = true

	room.Broadcast(room.GetGameState())
}

func (room *GameRoom) DetermineWinner() {
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
			// Сравниваем кикеры
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

	// Разделить банк между победителями
	prizePerWinner := room.Pot / len(winners)
	for _, idx := range winners {
		room.Players[idx].Chips += prizePerWinner
	}

	room.GamePhase = "finished"
	room.Broadcast(room.GetGameState())
}
