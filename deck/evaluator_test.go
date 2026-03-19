package deck

import (
	"testing"
)

func TestEvaluateHandRank(t *testing.T) {
	tests := []struct {
		name     string
		cards    []Card
		expected HandRank
	}{
		{
			"Royal Flush",
			[]Card{NewCard(Spades, 10), NewCard(Spades, 11), NewCard(Spades, 12), NewCard(Spades, 13), NewCard(Spades, 1)},
			StraightFlush,
		},
		{
			"Four of a Kind",
			[]Card{NewCard(Harts, 10), NewCard(Spades, 10), NewCard(Diamonds, 10), NewCard(Clubs, 10), NewCard(Spades, 2)},
			FourOfAKind,
		},
		{
			"Full House",
			[]Card{NewCard(Harts, 3), NewCard(Spades, 3), NewCard(Diamonds, 3), NewCard(Clubs, 4), NewCard(Spades, 4)},
			FullHouse,
		},
		{
			"Flush",
			[]Card{NewCard(Harts, 2), NewCard(Harts, 5), NewCard(Harts, 7), NewCard(Harts, 10), NewCard(Harts, 13)},
			Flush,
		},
		{
			"Straight",
			[]Card{NewCard(Harts, 5), NewCard(Spades, 6), NewCard(Diamonds, 7), NewCard(Clubs, 8), NewCard(Spades, 9)},
			Straight,
		},
		{
			"Ace-Low Straight",
			[]Card{NewCard(Harts, 1), NewCard(Spades, 2), NewCard(Diamonds, 3), NewCard(Clubs, 4), NewCard(Spades, 5)},
			Straight,
		},
		{
			"Three of a Kind",
			[]Card{NewCard(Harts, 7), NewCard(Spades, 7), NewCard(Diamonds, 7), NewCard(Clubs, 12), NewCard(Spades, 2)},
			ThreeOfAKind,
		},
		{
			"Two Pair",
			[]Card{NewCard(Harts, 8), NewCard(Spades, 8), NewCard(Diamonds, 4), NewCard(Clubs, 4), NewCard(Spades, 12)},
			TwoPair,
		},
		{
			"Pair",
			[]Card{NewCard(Harts, 9), NewCard(Spades, 9), NewCard(Diamonds, 4), NewCard(Clubs, 2), NewCard(Spades, 13)},
			Pair,
		},
		{
			"High Card",
			[]Card{NewCard(Harts, 1), NewCard(Spades, 10), NewCard(Diamonds, 7), NewCard(Clubs, 4), NewCard(Spades, 2)},
			HighCard,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hand := Evaluate(tc.cards)
			if hand.Rank != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, hand.Rank)
			}
		})
	}
}

func TestEvaluateTieBreaks(t *testing.T) {
	// A-K high vs A-Q high
	h1 := Evaluate([]Card{NewCard(Harts, 1), NewCard(Spades, 13), NewCard(Diamonds, 7), NewCard(Clubs, 4), NewCard(Spades, 2)})
	h2 := Evaluate([]Card{NewCard(Harts, 1), NewCard(Spades, 12), NewCard(Diamonds, 7), NewCard(Clubs, 4), NewCard(Spades, 2)})
	if h1.Score <= h2.Score {
		t.Errorf("A-K high should beat A-Q high")
	}

	// 8s trips with A kicker vs 8s trips with K kicker
	t1 := Evaluate([]Card{NewCard(Harts, 8), NewCard(Spades, 8), NewCard(Diamonds, 8), NewCard(Clubs, 1), NewCard(Spades, 2)})
	t2 := Evaluate([]Card{NewCard(Harts, 8), NewCard(Spades, 8), NewCard(Diamonds, 8), NewCard(Clubs, 13), NewCard(Spades, 2)})
	if t1.Score <= t2.Score {
		t.Errorf("trips with A kicker should beat trips with K kicker")
	}

	// 7 card evaluation -> A flush is formed
	cards7 := []Card{
		NewCard(Harts, 2), NewCard(Harts, 5), NewCard(Harts, 7), NewCard(Harts, 10), NewCard(Harts, 13),
		NewCard(Spades, 2), NewCard(Clubs, 2),
	}
	hand7 := Evaluate(cards7)
	if hand7.Rank != Flush {
		t.Errorf("best 5 cards should form a Flush, got %s", hand7.Rank)
	}
}
