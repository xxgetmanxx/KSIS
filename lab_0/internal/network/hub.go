package network

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"poker-duel/internal/game"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type Hub struct {
	Clients    map[*websocket.Conn]string
	Rooms      map[string]*GameRoom
	ArenaQueue []*websocket.Conn
	SpinQueue  []*websocket.Conn
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Clients: make(map[*websocket.Conn]string),
		Rooms:   make(map[string]*GameRoom),
	}
}

func (h *Hub) Run() {
	for {
		time.Sleep(2 * time.Second)
		h.mu.Lock()
		if len(h.ArenaQueue) >= 2 {
			p1, p2 := h.ArenaQueue[0], h.ArenaQueue[1]
			h.ArenaQueue = h.ArenaQueue[2:]
			h.startMatch("ARENA", p1, p2, 100, 200)
		}
		if len(h.SpinQueue) >= 2 {
			p1, p2 := h.SpinQueue[0], h.SpinQueue[1]
			h.SpinQueue = h.SpinQueue[2:]
			h.startMatch("SPIN", p1, p2, 100, 200)
		}
		h.mu.Unlock()
	}
}

func (h *Hub) startMatch(mode string, conn1, conn2 *websocket.Conn, smallBlind, bigBlind int) {
	var room *GameRoom
	if mode == "FRIEND" {
		room, _ = h.getPlayerRoom(conn1)
		for _, p := range room.Players {
			p.Cards = []game.Card{}
			p.Chips = 10000
			p.Bet = 0
			p.Folded = false
			p.IsTurn = false
			p.AllIn = false
		}
	} else {
		roomID := fmt.Sprintf("R_%d", rand.Intn(100000))
		room = &GameRoom{
			ID:         roomID,
			SmallBlind: smallBlind,
			BigBlind:   bigBlind,
			Hub:        h, // Set hub reference for timeout handling
		}
		room.Players = []*PlayerState{
			{
				Conn:   conn1,
				Name:   h.Clients[conn1],
				Cards:  []game.Card{},
				Chips:  10000,
				Bet:    0,
				Folded: false,
				IsTurn: false,
				AllIn:  false,
			},
			{
				Conn:   conn2,
				Name:   h.Clients[conn2],
				Cards:  []game.Card{},
				Chips:  10000,
				Bet:    0,
				Folded: false,
				IsTurn: false,
				AllIn:  false,
			},
		}
		h.Rooms[roomID] = room
	}
	room.Deck = []game.Card{}
	room.Table = []game.Card{}
	room.Pot = 0
	room.CurrentTurn = 0
	room.GamePhase = "waiting"
	room.DealerPos = 0
	if room.SmallBlind == 0 {
		room.SmallBlind = smallBlind
		room.BigBlind = bigBlind
	}

	multiplier := 2
	if mode == "SPIN" {
		multipliers := []int{2, 4, 6, 10}
		multiplier = multipliers[rand.Intn(4)]
	}

	// Отправка game_start
	msg1, _ := json.Marshal(map[string]interface{}{
		"type": "game_start", "room": room.ID, "mode": mode, "multiplier": multiplier,
	})
	msg2, _ := json.Marshal(map[string]interface{}{
		"type": "game_start", "room": room.ID, "mode": mode, "multiplier": multiplier,
	})

	room.Players[0].Conn.WriteMessage(websocket.TextMessage, msg1)
	room.Players[1].Conn.WriteMessage(websocket.TextMessage, msg2)

	// Запускаем игру и отправляем game_state
	room.StartGame()
}

func (h *Hub) getPlayerRoom(conn *websocket.Conn) (*GameRoom, int) {
	for _, room := range h.Rooms {
		for i, p := range room.Players {
			if p.Conn == conn {
				return room, i
			}
		}
	}
	return nil, -1
}

// processTimeout handles timeout actions with proper mutex locking
func (h *Hub) processTimeout(roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.Rooms[roomID]
	if !ok {
		return
	}

	// Check if the room is still active
	if room.GamePhase == "finished" || room.GamePhase == "showdown" {
		return
	}

	// Process the timeout action
	room.HandleTimeout()
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	username := r.URL.Query().Get("user")
	hub.mu.Lock()
	hub.Clients[conn] = username
	hub.mu.Unlock()

	go func() {
		defer func() {
			hub.mu.Lock()
			delete(hub.Clients, conn)
			conn.Close()
			hub.mu.Unlock()
		}()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var req map[string]interface{}
			json.Unmarshal(message, &req)

			hub.mu.Lock()
			
			if req["action"] == "search_arena" {
				hub.ArenaQueue = append(hub.ArenaQueue, conn)
			} else if req["action"] == "search_spin" {
				hub.SpinQueue = append(hub.SpinQueue, conn)
			} else if req["action"] == "create_friend" {
				code := req["code"].(string)
				room := &GameRoom{ID: code, GamePhase: "waiting", Hub: hub}
				room.Players = append(room.Players, &PlayerState{
					Conn:   conn,
					Name:   hub.Clients[conn],
					Cards:  []game.Card{},
					Chips:  10000,
					Bet:    0,
					Folded: false,
					IsTurn: false,
					AllIn:  false,
				})
				hub.Rooms[code] = room
			} else if req["action"] == "join_friend" {
				code := req["code"].(string)
				if room, ok := hub.Rooms[code]; ok && len(room.Players) < 2 {
					if len(room.Players) == 1 {
						room.Players = append(room.Players, &PlayerState{
							Conn:   conn,
							Name:   hub.Clients[conn],
							Cards:  []game.Card{},
							Chips:  10000,
							Bet:    0,
							Folded: false,
							IsTurn: false,
							AllIn:  false,
						})
						hub.startMatch("FRIEND", room.Players[0].Conn, room.Players[1].Conn, 50, 100)
					}
				}
			} else if req["action"] == "fold" {
				room, playerIdx := hub.getPlayerRoom(conn)
				if room != nil && playerIdx != -1 && room.Players[playerIdx].IsTurn {
					room.PlayerFold(playerIdx)
				}
			} else if req["action"] == "check" {
				room, playerIdx := hub.getPlayerRoom(conn)
				if room != nil && playerIdx != -1 && room.Players[playerIdx].IsTurn {
					room.PlayerCheck(playerIdx)
				}
			} else if req["action"] == "call" {
				room, playerIdx := hub.getPlayerRoom(conn)
				if room != nil && playerIdx != -1 && room.Players[playerIdx].IsTurn {
					room.PlayerCall(playerIdx)
				}
			} else if req["action"] == "bet" {
				room, playerIdx := hub.getPlayerRoom(conn)
				if room != nil && playerIdx != -1 && room.Players[playerIdx].IsTurn {
					amount := 0
					if amt, ok := req["amount"].(float64); ok {
						amount = int(amt)
					}
					room.PlayerBet(playerIdx, amount)
				}
			} else if req["action"] == "raise" {
				room, playerIdx := hub.getPlayerRoom(conn)
				if room != nil && playerIdx != -1 && room.Players[playerIdx].IsTurn {
					amount := 0
					if amt, ok := req["amount"].(float64); ok {
						amount = int(amt)
					}
					room.PlayerRaise(playerIdx, amount)
				}
			}
			
			hub.mu.Unlock()
		}
	}()
}
