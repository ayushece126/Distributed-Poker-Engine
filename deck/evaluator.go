package deck

import (
	"sort"
)

type HandRank int

const (
	HighCard HandRank = iota
	Pair
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	FullHouse
	FourOfAKind
	StraightFlush
)

func (h HandRank) String() string {
	names := []string{"High Card", "Pair", "Two Pair", "Three of a Kind", "Straight", "Flush", "Full House", "Four of a Kind", "Straight Flush"}
	if int(h) < len(names) {
		return names[h]
	}
	return "Unknown"
}

type Hand struct {
	Rank  HandRank
	Cards []Card
	Score int64 // Used to definitively break ties
}

// Evaluate finds the best 5-card combination from the provided cards.
func Evaluate(cards []Card) Hand {
	if len(cards) < 5 {
		panic("need at least 5 cards to evaluate")
	}

	bestHand := Hand{Score: -1}
	combos := getCombinations(cards, 5)

	for _, combo := range combos {
		h := evaluate5(combo)
		if h.Score > bestHand.Score {
			bestHand = h
		}
	}
	return bestHand
}

func getCombinations(set []Card, k int) [][]Card {
	var subsets [][]Card
	var recurse func(int, []Card)
	recurse = func(start int, current []Card) {
		if len(current) == k {
			cp := make([]Card, k)
			copy(cp, current)
			subsets = append(subsets, cp)
			return
		}
		for i := start; i < len(set); i++ {
			recurse(i+1, append(current, set[i]))
		}
	}
	recurse(0, []Card{})
	return subsets
}

// evaluate5 statically evaluates exactly 5 cards
func evaluate5(cards []Card) Hand {
	sorted := make([]Card, 5)
	copy(sorted, cards)
	
	// Sort by value descending, treating A as 14
	sort.Slice(sorted, func(i, j int) bool {
		vi, vj := sorted[i].Value, sorted[j].Value
		if vi == 1 { vi = 14 }
		if vj == 1 { vj = 14 }
		return vi > vj
	})

	isFlush := true
	for i := 1; i < 5; i++ {
		if sorted[i].Suit != sorted[0].Suit {
			isFlush = false
			break
		}
	}

	isStraight := true
	for i := 1; i < 5; i++ {
		vi, vj := sorted[i-1].Value, sorted[i].Value
		if vi == 1 { vi = 14 }
		if vj == 1 { vj = 14 }
		if vi-vj != 1 {
			isStraight = false
			break
		}
	}

	// Handle Ace-low straight A-5-4-3-2
	isAceLowStraight := false
	if !isStraight {
		v0 := sorted[0].Value
		if v0 == 1 { v0 = 14 }
		if v0 == 14 && sorted[1].Value == 5 && sorted[2].Value == 4 && sorted[3].Value == 3 && sorted[4].Value == 2 {
			isStraight = true
			isAceLowStraight = true
			// Reorder to 5-4-3-2-A
			sorted = []Card{sorted[1], sorted[2], sorted[3], sorted[4], sorted[0]}
		}
	}

	counts := make(map[int]int)
	for _, c := range sorted {
		v := c.Value
		if v == 1 { v = 14 }
		counts[v]++
	}

	hasQuad, hasTrip := false, false
	pairs := 0
	for _, count := range counts {
		if count == 4 { hasQuad = true }
		if count == 3 { hasTrip = true }
		if count == 2 { pairs++ }
	}

	var rank HandRank
	if isStraight && isFlush {
		rank = StraightFlush
	} else if hasQuad {
		rank = FourOfAKind
	} else if hasTrip && pairs == 1 {
		rank = FullHouse
	} else if isFlush {
		rank = Flush
	} else if isStraight {
		rank = Straight
	} else if hasTrip {
		rank = ThreeOfAKind
	} else if pairs == 2 {
		rank = TwoPair
	} else if pairs == 1 {
		rank = Pair
	} else {
		rank = HighCard
	}

	// Re-sort cards by their frequency of occurrence, then naturally by value
	// This puts the most critical parts of the combination (like a Quad or Trip) at the front for tie-breakers
	finalSorted := make([]Card, 5)
	copy(finalSorted, sorted)
	sort.Slice(finalSorted, func(i, j int) bool {
		vi, vj := finalSorted[i].Value, finalSorted[j].Value
		if vi == 1 { vi = 14 }
		if vj == 1 { vj = 14 }
		ci, cj := counts[vi], counts[vj]
		if ci != cj {
			return ci > cj
		}
		return vi > vj
	})

	var score int64 = int64(rank) << 20
	// 4 bits per card
	for i, c := range finalSorted {
		v := c.Value
		if v == 1 && !(isAceLowStraight && i == 4) { v = 14 }
		if isAceLowStraight && i == 4 { v = 1 } // Ace is low in Ace-2-3-4-5
		score |= int64(v) << (4 * (4 - i))
	}

	return Hand{
		Rank:  rank,
		Cards: finalSorted,
		Score: score,
	}
}
