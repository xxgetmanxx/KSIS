package game

import (
	"math/rand"
	"time"
)

type Card struct {
	Suit  string `json:"suit"`  
	Value string `json:"value"` 
	Score int    `json:"score"` 
}

func NewDeck() []Card {
	suits := []string{"♠", "♥", "♦", "♣"}
	values := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}

	deck := make([]Card, 0, 52)
	for _, s := range suits {
		for i, v := range values {
			deck = append(deck, Card{Suit: s, Value: v, Score: i + 2})
		}
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}
