package p2p

import "github.com/anthdm/ggpoker/deck"

type Message struct {
	Payload any
	From    string
}

type BroadcastTo struct {
	To      []string
	Payload any
}

func NewMessage(from string, payload any) *Message {
	return &Message{
		From:    from,
		Payload: payload,
	}
}

type Handshake struct {
	Version     string
	GameVariant GameVariant
	GameStatus  GameStatus
	ListenAddr  string
}

type MessagePlayerAction struct {
	// CurrentGameStatus is the current status of the sending player his game.
	// this needs to the exact same as ours.
	CurrentGameStatus GameStatus
	// Action is the action that the player is willin to take.
	Action PlayerAction
	// The value of the bet if any
	Value int
}

type MessagePreFlop struct{}

func (msg MessagePreFlop) String() string {
	return "MSG: PREFLOP"
}

type MessagePeerList struct {
	Peers []string
}

type MessageEncDeck struct {
	Deck [][]byte
}

type MessageDeckFinal struct {
	Deck [][]byte
}

type MessagePassUnlockCard struct {
	RingID     string // Unique ID to distinguish concurrent rings
	InitNode   string
	TargetNode string
	Indexes    []int
	Payloads   [][]byte
}

type MessageRevealCommunity struct {
	Cards []deck.Card
}

type MessageShowHand struct {
	Addr  string
	Cards  []deck.Card
}

type MessageReady struct{}

func (msg MessageReady) String() string {
	return "MSG: READY"
}
