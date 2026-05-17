package game

import "math/rand"

// GetBotAction генерирует действие бота в зависимости от уровня сложности
func GetBotAction(difficulty string, botHand []Card, tableCards []Card) string {
	score := 0
	for _, c := range botHand {
		score += c.Score
	}

	switch difficulty {
	case "EASY":
		if rand.Float64() < 0.15 {
			return "fold"
		}
		return "check"

	case "MEDIUM":
		if score < 10 {
			return "fold"
		}
		if score > 22 {
			return "raise"
		}
		return "check"

	case "HARD":
		if score > 24 {
			return "raise"
		}
		hasHighCardOnTable := false
		for _, tc := range tableCards {
			if tc.Value == "A" || tc.Value == "K" {
				hasHighCardOnTable = true
			}
		}
		if hasHighCardOnTable && rand.Float64() < 0.20 {
			return "raise" // Симуляция блефа при опасных картах на борде
		}
		if score < 14 {
			return "fold"
		}
		return "check"
	}

	return "check"
}
