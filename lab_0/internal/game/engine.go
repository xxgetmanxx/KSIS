package game

import "sort"


const (
	HighCard = iota
	OnePair
	TwoPair
	ThreeOfKind
	Straight
	Flush
	FullHouse
	FourOfKind
	StraightFlush
	RoyalFlush
)


func EvaluateHand(hand []Card, table []Card) (int, []int) {
	allCards := append(hand, table...)
	if len(allCards) < 5 {
		return HighCard, getSortedScores(allCards)
	}

	
	if rank, kickers := checkRoyalFlush(allCards); rank != -1 {
		return rank, kickers
	}
	if rank, kickers := checkStraightFlush(allCards); rank != -1 {
		return rank, kickers
	}
	if rank, kickers := checkFourOfKind(allCards); rank != -1 {
		return rank, kickers
	}
	if rank, kickers := checkFullHouse(allCards); rank != -1 {
		return rank, kickers
	}
	if rank, kickers := checkFlush(allCards); rank != -1 {
		return rank, kickers
	}
	if rank, kickers := checkStraight(allCards); rank != -1 {
		return rank, kickers
	}
	if rank, kickers := checkThreeOfKind(allCards); rank != -1 {
		return rank, kickers
	}
	if rank, kickers := checkTwoPair(allCards); rank != -1 {
		return rank, kickers
	}
	if rank, kickers := checkOnePair(allCards); rank != -1 {
		return rank, kickers
	}

	return HighCard, getSortedScores(allCards)
}


func EvaluateWinner(p1Hand, p2Hand, table []Card) int {
	rank1, kickers1 := EvaluateHand(p1Hand, table)
	rank2, kickers2 := EvaluateHand(p2Hand, table)

	if rank1 > rank2 {
		return 1
	}
	if rank2 > rank1 {
		return 2
	}

	
	for i := 0; i < len(kickers1) && i < len(kickers2); i++ {
		if kickers1[i] > kickers2[i] {
			return 1
		}
		if kickers2[i] > kickers1[i] {
			return 2
		}
	}
	return 0
}

func getSortedScores(cards []Card) []int {
	scores := make([]int, len(cards))
	for i, c := range cards {
		scores[i] = c.Score
	}
	sort.Sort(sort.Reverse(sort.IntSlice(scores)))
	return scores
}

func checkRoyalFlush(cards []Card) (int, []int) {
	rank, kickers := checkStraightFlush(cards)
	if rank != -1 && kickers[0] == 14 { 
		return RoyalFlush, kickers
	}
	return -1, nil
}

func checkStraightFlush(cards []Card) (int, []int) {
	suitGroups := make(map[string][]Card)
	for _, c := range cards {
		suitGroups[c.Suit] = append(suitGroups[c.Suit], c)
	}

	for _, suitCards := range suitGroups {
		if len(suitCards) >= 5 {
			if rank, kickers := checkStraight(suitCards); rank != -1 {
				return StraightFlush, kickers
			}
		}
	}
	return -1, nil
}

func checkFourOfKind(cards []Card) (int, []int) {
	counts := getScoreCounts(cards)
	for score, count := range counts {
		if count == 4 {
			kickers := []int{score}
			for s := range counts {
				if s != score {
					kickers = append(kickers, s)
					break
				}
			}
			return FourOfKind, kickers
		}
	}
	return -1, nil
}

func checkFullHouse(cards []Card) (int, []int) {
	counts := getScoreCounts(cards)
	var three, pair int
	for score, count := range counts {
		if count == 3 {
			three = score
		} else if count >= 2 {
			pair = score
		}
	}
	if three != 0 && pair != 0 {
		return FullHouse, []int{three, pair}
	}
	return -1, nil
}

func checkFlush(cards []Card) (int, []int) {
	suitGroups := make(map[string][]Card)
	for _, c := range cards {
		suitGroups[c.Suit] = append(suitGroups[c.Suit], c)
	}
	for _, suitCards := range suitGroups {
		if len(suitCards) >= 5 {
			return Flush, getSortedScores(suitCards)[:5]
		}
	}
	return -1, nil
}

func checkStraight(cards []Card) (int, []int) {
	scores := make(map[int]bool)
	for _, c := range cards {
		scores[c.Score] = true
	}

	
	if scores[14] && scores[2] && scores[3] && scores[4] && scores[5] {
		return Straight, []int{5}
	}

	for high := 14; high >= 5; high-- {
		allPresent := true
		for i := 0; i < 5; i++ {
			if !scores[high-i] {
				allPresent = false
				break
			}
		}
		if allPresent {
			return Straight, []int{high}
		}
	}
	return -1, nil
}

func checkThreeOfKind(cards []Card) (int, []int) {
	counts := getScoreCounts(cards)
	for score, count := range counts {
		if count == 3 {
			kickers := []int{score}
			others := []int{}
			for s := range counts {
				if s != score {
					others = append(others, s)
				}
			}
			sort.Sort(sort.Reverse(sort.IntSlice(others)))
			kickers = append(kickers, others[:2]...)
			return ThreeOfKind, kickers
		}
	}
	return -1, nil
}

func checkTwoPair(cards []Card) (int, []int) {
	counts := getScoreCounts(cards)
	pairs := []int{}
	for score, count := range counts {
		if count == 2 {
			pairs = append(pairs, score)
		}
	}
	if len(pairs) >= 2 {
		sort.Sort(sort.Reverse(sort.IntSlice(pairs)))
		kicker := 0
		for s := range counts {
			if s != pairs[0] && s != pairs[1] {
				if s > kicker {
					kicker = s
				}
			}
		}
		return TwoPair, []int{pairs[0], pairs[1], kicker}
	}
	return -1, nil
}

func checkOnePair(cards []Card) (int, []int) {
	counts := getScoreCounts(cards)
	for score, count := range counts {
		if count == 2 {
			kickers := []int{score}
			others := []int{}
			for s := range counts {
				if s != score {
					others = append(others, s)
				}
			}
			sort.Sort(sort.Reverse(sort.IntSlice(others)))
			kickers = append(kickers, others[:3]...)
			return OnePair, kickers
		}
	}
	return -1, nil
}

func getScoreCounts(cards []Card) map[int]int {
	counts := make(map[int]int)
	for _, c := range cards {
		counts[c.Score]++
	}
	return counts
}
