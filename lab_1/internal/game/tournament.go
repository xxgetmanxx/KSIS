package game

import "fmt"

type Match struct {
	Player1 string
	Player2 string
	Winner  string
}

type TournamentBracket struct {
	ID           string
	Round        int
	QuarterFinal []Match
	SemiFinal    []Match
	Final        Match
	IsFinished   bool
}

func CreateTournament(players []string) *TournamentBracket {
	for len(players) < 8 {
		players = append(players, fmt.Sprintf("Bot_%d", len(players)+1))
	}

	t := &TournamentBracket{
		ID:    "TOURNEY_GLASS_1",
		Round: 1,
	}

	for i := 0; i < 8; i += 2 {
		t.QuarterFinal = append(t.QuarterFinal, Match{
			Player1: players[i],
			Player2: players[i+1],
		})
	}
	return t
}
