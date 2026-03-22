package p2p

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anthdm/ggpoker/deck"
	"github.com/sirupsen/logrus"
)

type PlayersList struct {
	lock sync.RWMutex
	list []string
}

func NewPlayersList() *PlayersList {
	return &PlayersList{
		list: []string{},
	}
}

func (p *PlayersList) List() []string {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.list
}

func (p *PlayersList) len() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return len(p.list)
}

func (p *PlayersList) add(addr string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.list = append(p.list, addr)
	sort.Sort(p)
}

func (p *PlayersList) remove(addr string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for i, v := range p.list {
		if v == addr {
			p.list = append(p.list[:i], p.list[i+1:]...)
			break
		}
	}
}

func (p *PlayersList) getIndex(addr string) int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for i := 0; i < len(p.list); i++ {
		if addr == p.list[i] {
			return i
		}
	}
	return -1
}

func (p *PlayersList) get(index any) string {
	p.lock.RLock()
	defer p.lock.RUnlock()

	var i int
	switch v := index.(type) {
	case int:
		i = v
	case int32:
		i = int(v)
	}

	if len(p.list)-1 < i {
		panic("the given index is too high")
	}

	return p.list[i]
}

func (p *PlayersList) Len() int { return len(p.list) }
func (p *PlayersList) Swap(i, j int) {
	p.list[i], p.list[j] = p.list[j], p.list[i]
}
func (p *PlayersList) Less(i, j int) bool {
	portI, _ := strconv.Atoi(p.list[i][1:])
	portJ, _ := strconv.Atoi(p.list[j][1:])

	return portI < portJ
}

type AtomicInt struct {
	value int32
}

func NewAtomicInt(value int32) *AtomicInt {
	return &AtomicInt{
		value: value,
	}
}

func (a *AtomicInt) String() string {
	return fmt.Sprintf("%d", a.value)
}

func (a *AtomicInt) Set(value int32) {
	atomic.StoreInt32(&a.value, value)
}

func (a *AtomicInt) Get() int32 {
	return atomic.LoadInt32(&a.value)
}

func (a *AtomicInt) Inc() {
	atomic.AddInt32(&a.value, 1)
}

type PlayerActionsRecv struct {
	mu          sync.RWMutex
	recvActions map[string]MessagePlayerAction
}

func NewPlayerActionsRevc() *PlayerActionsRecv {
	return &PlayerActionsRecv{
		recvActions: make(map[string]MessagePlayerAction),
	}
}

func (pa *PlayerActionsRecv) addAction(from string, action MessagePlayerAction) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.recvActions[from] = action
}

func (pa *PlayerActionsRecv) clear() {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.recvActions = map[string]MessagePlayerAction{}
}

// TODO: (@anthdm) Maybe use playersList instead??
type PlayersReady struct {
	mu           sync.RWMutex
	recvStatutes map[string]bool
}

func NewPlayersReady() *PlayersReady {
	return &PlayersReady{
		recvStatutes: make(map[string]bool),
	}
}

func (pr *PlayersReady) addRecvStatus(from string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.recvStatutes[from] = true
}

func (pr *PlayersReady) len() int {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	return len(pr.recvStatutes)
}

func (pr *PlayersReady) clear() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.recvStatutes = make(map[string]bool)
}

type Game struct {
	listenAddr  string
	broadcastch chan BroadcastTo

	// currentStatus should be atomically accessable.
	currentStatus *AtomicInt

	// currentPlayerAction should be atomically accessable.
	currentPlayerAction *AtomicInt
	// currentDealer should be atomically accessable.
	// NOTE: this will be -1 when the game is in a bootstrapped state.
	currentDealer *AtomicInt
	// currentPlayerTurn should be atomically accessable.
	currentPlayerTurn *AtomicInt

	playersReady      *PlayersReady
	recvPlayerActions *PlayerActionsRecv

	// playersList is the list of connected players to the network
	playersList *PlayersList

	table *Table
	Pot   int

	// Mental Poker State
	Keys           *KeyPair
	EncryptedDeck  [][]byte
	DeckPointer    int
	HoleCards      []deck.Card                // Our private hole cards
	CommunityCards []deck.Card                // Shared community cards
	ShowdownHands  map[string][]deck.Card     // addr -> revealed hole cards from showdown
	ringCounter    int                        // increments to generate unique ring IDs
}

func NewGame(addr string, bc chan BroadcastTo) *Game {
	keys, err := GenerateKeys()
	if err != nil {
		panic(fmt.Sprintf("failed to generate Mental Poker keys: %s", err))
	}

	g := &Game{
		listenAddr:          addr,
		broadcastch:         bc,
		currentStatus:       NewAtomicInt(int32(GameStatusConnected)),
		playersReady:        NewPlayersReady(),
		playersList:         NewPlayersList(),
		currentPlayerAction: NewAtomicInt(0),
		currentDealer:       NewAtomicInt(0),
		recvPlayerActions:   NewPlayerActionsRevc(),
		currentPlayerTurn:   NewAtomicInt(0),
		table:               NewTable(6),
		Keys:                keys,
		ShowdownHands:       make(map[string][]deck.Card),
	}

	g.playersList.add(addr)

	go g.loop()

	return g
}

func (g *Game) canTakeAction(from string) bool {
	currentPlayerAddr := g.playersList.get(g.currentPlayerTurn.Get())
	return currentPlayerAddr == from
}

func (g *Game) isFromCurrentDealer(from string) bool {
	return g.playersList.get(g.currentDealer.Get()) == from
}

func (g *Game) applyPlayerAction(addr string, action PlayerAction, value int) error {
	player, err := g.table.GetPlayer(addr)
	if err != nil {
		return err
	}

	if action == PlayerActionBet || action == PlayerActionCall {
		if player.Balance < value {
			return fmt.Errorf("player %s has insufficient balance for bet of %d", addr, value)
		}
		player.Balance -= value
		player.CurrentBet += value
		g.Pot += value
	}
	return nil
}

func (g *Game) handlePlayerAction(from string, action MessagePlayerAction) error {
	if !g.canTakeAction(from) {
		return fmt.Errorf("player (%s) taking action before his turn", from)
	}

	if action.CurrentGameStatus != GameStatus(g.currentStatus.Get()) && !g.isFromCurrentDealer(from) {
		return fmt.Errorf("player (%s) has not the correct game status (%s)", from, action.CurrentGameStatus)
	}

	if err := g.applyPlayerAction(from, action.Action, action.Value); err != nil {
		return err
	}

	g.recvPlayerActions.addAction(from, action)

	if g.playersList.get(g.currentDealer.Get()) == from {
		g.advanceToNexRound() // Automatically steps if the dealer acts
	}

	g.incNextPlayer()

	logrus.WithFields(logrus.Fields{
		"we":     g.listenAddr,
		"from":   from,
		"action": action,
	}).Info("recv player action")

	return nil
}

func (g *Game) TakeAction(action PlayerAction, value int) error {
	if !g.canTakeAction(g.listenAddr) {
		return fmt.Errorf("taking action before its my turn %s", g.listenAddr)
	}

	if err := g.applyPlayerAction(g.listenAddr, action, value); err != nil {
		return err
	}

	g.currentPlayerAction.Set((int32)(action))

	g.incNextPlayer()

	if g.listenAddr == g.playersList.get(g.currentDealer.Get()) {
		g.advanceToNexRound() // Automatically steps if the dealer acts
	}

	a := MessagePlayerAction{
		Action:            action,
		CurrentGameStatus: GameStatus(g.currentStatus.Get()),
		Value:             value,
	}
	g.sendToPlayers(a, g.getOtherPlayers()...)

	return nil
}

func (g *Game) getNextGameStatus() GameStatus {
	status := GameStatus(g.currentStatus.Get())
	switch status {
	case GameStatusPreFlop:
		return GameStatusFlop
	case GameStatusFlop:
		return GameStatusTurn
	case GameStatusTurn:
		return GameStatusRiver
	case GameStatusRiver:
		return GameStatusPlayerReady
	default:
		fmt.Printf("invalid status => %+v\n", status)
		panic("invalid game status")
	}
}

func (g *Game) advanceToNexRound() {
	g.recvPlayerActions.clear()
	g.currentPlayerAction.Set(int32(PlayerActionNone))

	if GameStatus(g.currentStatus.Get()) == GameStatusRiver {
		g.Showdown()
		return
	}

	nextStatus := g.getNextGameStatus()
	g.currentStatus.Set(int32(nextStatus))

	// Only the dealer initiates the community card sequential unspooling ring
	_, isDealer := g.getCurrentDealerAddr()
	if isDealer {
		var indexes []int
		if nextStatus == GameStatusFlop {
			g.DeckPointer++ // Burn 1 card
			indexes = []int{g.DeckPointer, g.DeckPointer + 1, g.DeckPointer + 2}
			g.DeckPointer += 3
		} else if nextStatus == GameStatusTurn || nextStatus == GameStatusRiver {
			g.DeckPointer++ // Burn 1 card
			indexes = []int{g.DeckPointer}
			g.DeckPointer++
		}

		if len(indexes) > 0 {
			g.ringCounter++
			ringID := fmt.Sprintf("%s-community-%d", g.listenAddr, g.ringCounter)

			logrus.Infof("[%s] Dealer initiating community sequential unspooling for %v", g.listenAddr, indexes)
			payloads := make([][]byte, len(indexes))
			for i, idx := range indexes {
				payloads[i] = g.EncryptedDeck[idx]
			}
			nextPlayer, _ := g.table.GetPlayerAfter(g.listenAddr)
			g.sendToPlayers(MessagePassUnlockCard{
				RingID:     ringID,
				InitNode:   g.listenAddr,
				TargetNode: "COMMUNITY",
				Indexes:    indexes,
				Payloads:   payloads,
			}, nextPlayer.addr)
		}
	} else {
		// Non-dealers sync their pointer to stay aligned: burn 1 + deal N
		if nextStatus == GameStatusFlop {
			g.DeckPointer += 4 // Burn 1 + Flop 3
		} else if nextStatus == GameStatusTurn || nextStatus == GameStatusRiver {
			g.DeckPointer += 2 // Burn 1 + Deal 1
		}
	}
}

func (g *Game) Showdown() {
	logrus.WithFields(logrus.Fields{
		"pot": g.Pot,
		"we":  g.listenAddr,
	}).Info("entering showdown")

	// Step 1: Broadcast our hole cards so every node can independently verify
	g.sendToPlayers(MessageShowHand{
		Addr:  g.listenAddr,
		Cards: g.HoleCards,
	}, g.getOtherPlayers()...)

	// Store our own hand into the ShowdownHands map for evaluation
	g.ShowdownHands[g.listenAddr] = g.HoleCards

	// Step 2: Evaluate locally once all hands are collected
	// (In a real implementation, we would wait for all MessageShowHand messages first.
	// For now, this evaluates with whatever hands we have.)
	go func() {
		// Wait a moment for all ShowHand messages to arrive
		time.Sleep(3 * time.Second)
		g.evaluateShowdown()
	}()
}

func (g *Game) HandleShowHand(from string, msg MessageShowHand) error {
	g.ShowdownHands[msg.Addr] = msg.Cards
	logrus.Infof("[%s] Received showdown hand from %s: %v", g.listenAddr, msg.Addr, msg.Cards)
	return nil
}

func (g *Game) evaluateShowdown() {
	if len(g.CommunityCards) < 5 {
		logrus.Warnf("[%s] Showdown called with only %d community cards", g.listenAddr, len(g.CommunityCards))
	}

	var bestScore int64
	var winnerAddr string

	for addr, holeCards := range g.ShowdownHands {
		if len(holeCards) < 2 {
			continue
		}
		// Combine 2 hole cards + 5 community cards for evaluation
		allCards := make([]deck.Card, 0, 7)
		allCards = append(allCards, holeCards...)
		allCards = append(allCards, g.CommunityCards...)
		result := deck.Evaluate(allCards)

		logrus.Infof("[%s] Player %s has hand: %s (score: %d)", g.listenAddr, addr, result.Rank, result.Score)

		if result.Score > bestScore {
			bestScore = result.Score
			winnerAddr = addr
		}
	}

	if winnerAddr == "" {
		logrus.Warnf("[%s] No valid hands found in showdown", g.listenAddr)
		return
	}

	winner, err := g.table.GetPlayer(winnerAddr)
	if err != nil {
		logrus.Errorf("[%s] Could not find winner %s on table: %s", g.listenAddr, winnerAddr, err)
		return
	}

	winner.Balance += g.Pot
	logrus.Infof("[%s] Showdown winner: %s with score %d. Pot %d awarded. New balance: %d",
		g.listenAddr, winnerAddr, bestScore, g.Pot, winner.Balance)
	g.Pot = 0

	// Reset for next hand
	g.HoleCards = nil
	g.CommunityCards = nil
	g.ShowdownHands = make(map[string][]deck.Card)
	g.EncryptedDeck = nil
	g.DeckPointer = 0
	g.ringCounter = 0

	g.currentDealer.Set(int32(g.getNextDealer()))
	g.SetReady()
}

func (g *Game) incNextPlayer() {
	if g.playersList.len()-1 == int(g.currentPlayerTurn.Get()) {
		g.currentPlayerTurn.Set(0)
		return
	}

	g.currentPlayerTurn.Inc()
}

func (g *Game) SetStatus(s GameStatus) {
	g.setStatus(s)
	g.table.SetPlayerStatus(g.listenAddr, s)
}

func (g *Game) setStatus(s GameStatus) {
	if s == GameStatusPreFlop {
		g.incNextPlayer()
	}

	// Only update the status when the status is different.
	if GameStatus(g.currentStatus.Get()) != s {
		g.currentStatus.Set(int32(s))
	}
}

func (g *Game) getCurrentDealerAddr() (string, bool) {
	currentDealerAddr := g.playersList.get(g.currentDealer.Get())

	return currentDealerAddr, g.listenAddr == currentDealerAddr
}

func (g *Game) LockAndShuffle(input [][]byte) [][]byte {
	out := make([][]byte, len(input))
	copy(out, input)

	// Fisher-Yates shuffle FIRST, then encrypt each card
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(out) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		out[i], out[j] = out[j], out[i]
	}

	// Encrypt every card with our key
	for i := range out {
		out[i] = Encrypt(out[i], g.Keys.EncryptKey)
	}
	return out
}

// nextRingID generates a unique ring identifier to prevent collisions between concurrent rings.
func (g *Game) nextRingID(label string) string {
	g.ringCounter++
	return fmt.Sprintf("%s-%s-%d", g.listenAddr, label, g.ringCounter)
}

func (g *Game) HandleMessageDeckFinal(from string, msg MessageDeckFinal) error {
	g.EncryptedDeck = msg.Deck
	g.DeckPointer = 0

	g.setStatus(GameStatusPreFlop)
	g.table.SetPlayerStatus(g.listenAddr, GameStatusPreFlop)

	logrus.Infof("[%s] Received final locked deck! Transitioning to PreFlop.", g.listenAddr)

	myIdx := g.playersList.getIndex(g.listenAddr)
	indexes := []int{myIdx * 2, myIdx*2 + 1}
	g.DeckPointer += g.playersList.len() * 2

	payloads := make([][]byte, len(indexes))
	for i, idx := range indexes {
		payloads[i] = g.EncryptedDeck[idx]
	}

	ringID := g.nextRingID("hole")
	nextPlayer, _ := g.table.GetPlayerAfter(g.listenAddr)
	g.sendToPlayers(MessagePassUnlockCard{
		RingID:     ringID,
		InitNode:   g.listenAddr,
		TargetNode: g.listenAddr,
		Indexes:    indexes,
		Payloads:   payloads,
	}, nextPlayer.addr)

	return nil
}

func (g *Game) HandlePassUnlockCard(from string, msg MessagePassUnlockCard) error {
	// Mathematically remove our layer of encryption
	for i := range msg.Payloads {
		msg.Payloads[i] = Decrypt(msg.Payloads[i], g.Keys.DecryptKey)
	}

	// Has the package travelled all the way around the ring back to the initiator?
	if msg.InitNode == g.listenAddr {
		resolvedCards := make([]deck.Card, 0, len(msg.Indexes))
		for i, idx := range msg.Indexes {
			finalBytes := msg.Payloads[i]
			if len(finalBytes) > 0 {
				card := IntToCard(int(finalBytes[len(finalBytes)-1]))
				resolvedCards = append(resolvedCards, card)
				logrus.Infof("[%s] Ring %s resolved card at index %d: %v", g.listenAddr, msg.RingID, idx, card)
			}
		}

		if msg.TargetNode == g.listenAddr {
			// These are our private hole cards — store them securely
			g.HoleCards = resolvedCards
			logrus.Infof("[%s] Stored private hole cards: %v", g.listenAddr, g.HoleCards)
		} else if msg.TargetNode == "COMMUNITY" {
			// Community cards resolved — broadcast to all peers
			g.CommunityCards = append(g.CommunityCards, resolvedCards...)
			logrus.Infof("[%s] Stored community cards (total %d): %v", g.listenAddr, len(g.CommunityCards), g.CommunityCards)
			g.sendToPlayers(MessageRevealCommunity{Cards: resolvedCards}, g.getOtherPlayers()...)
		}
		return nil
	}

	// The ring continues — pass to the next player
	nextPlayer, _ := g.table.GetPlayerAfter(g.listenAddr)
	g.sendToPlayers(msg, nextPlayer.addr)
	return nil
}

func (g *Game) HandleRevealCommunity(from string, msg MessageRevealCommunity) error {
	g.CommunityCards = append(g.CommunityCards, msg.Cards...)
	logrus.Infof("[%s] Received community cards (total %d): %v", g.listenAddr, len(g.CommunityCards), g.CommunityCards)
	return nil
}

func (g *Game) ShuffleAndEncrypt(from string, deck [][]byte) error {
	prevPlayer, err := g.table.GetPlayerBefore(g.listenAddr)
	if err != nil {
		panic(err)
	}
	if from != prevPlayer.addr {
		return fmt.Errorf("received encrypted deck from the wrong player (%s) should be (%s)", from, prevPlayer.addr)
	}

	_, isDealer := g.getCurrentDealerAddr()
	if isDealer && from == prevPlayer.addr {
		g.EncryptedDeck = deck
		g.DeckPointer = 0

		logrus.Infof("[%s] Mathematical Ring Loop Complete. Broadcasting finalized deck FIRST.", g.listenAddr)
		g.sendToPlayers(MessageDeckFinal{Deck: deck}, g.getOtherPlayers()...)

		// NOW change local stage
		g.setStatus(GameStatusPreFlop)
		g.table.SetPlayerStatus(g.listenAddr, GameStatusPreFlop)

		myIdx := g.playersList.getIndex(g.listenAddr)
		indexes := []int{myIdx * 2, myIdx*2 + 1}
		g.DeckPointer += g.playersList.len() * 2

		// Initiate unspooling for the dealer's own hole cards
		ringID := g.nextRingID("hole")
		payloads := make([][]byte, len(indexes))
		for i, idx := range indexes {
			payloads[i] = g.EncryptedDeck[idx]
		}
		nextPlayer, _ := g.table.GetPlayerAfter(g.listenAddr)
		g.sendToPlayers(MessagePassUnlockCard{
			RingID:     ringID,
			InitNode:   g.listenAddr,
			TargetNode: g.listenAddr,
			Indexes:    indexes,
			Payloads:   payloads,
		}, nextPlayer.addr)
		
		return nil
	}

	dealToPlayer, err := g.table.GetPlayerAfter(g.listenAddr)
	if err != nil {
		panic(err)
	}

	logrus.WithFields(logrus.Fields{
		"recvFromPlayer":  from,
		"we":              g.listenAddr,
		"dealingToPlayer": dealToPlayer.addr,
	}).Info("received cards and independently encrypting layer for next player")

	locked := g.LockAndShuffle(deck)

	g.sendToPlayers(MessageEncDeck{Deck: locked}, dealToPlayer.addr)
	g.setStatus(GameStatusDealing)

	return nil
}

func (g *Game) InitiateShuffleAndDeal() {
	dealToPlayer, err := g.table.GetPlayerAfter(g.listenAddr)
	if err != nil {
		panic(err)
	}

	g.setStatus(GameStatusDealing)

	plain := GeneratePlaintextDeck()
	locked := g.LockAndShuffle(plain)

	logrus.WithFields(logrus.Fields{
		"we": g.listenAddr,
		"to": dealToPlayer.addr,
	}).Info("Generated native Galois deck array. Passing to first node for encryption layer.")

	g.sendToPlayers(MessageEncDeck{Deck: locked}, dealToPlayer.addr)
}

func (g *Game) maybeDeal() {
	if GameStatus(g.currentStatus.Get()) == GameStatusPlayerReady {
		g.InitiateShuffleAndDeal()
	}
}

// SetPlayerReady is getting called when we receive a ready message
// from a player in the network.
func (g *Game) SetPlayerReady(addr string) {
	tablePos := g.playersList.getIndex(addr)
	g.table.AddPlayerOnPosition(addr, tablePos)

	// TODO: (@anthdm) check if we really need this??
	g.playersReady.addRecvStatus(addr)

	// TODO(@anthdm): This potentially going to cause an issue!
	// If we don't have enough players the round cannot be started.
	if g.table.LenPlayers() < 2 {
		return
	}

	// we need to check if we are the dealer of the current round.
	if _, areWeDealer := g.getCurrentDealerAddr(); areWeDealer {
		go func() {
			// if the game can start we will wait another
			// N amount of seconds to actually start dealing
			time.Sleep(5 * time.Second)
			g.maybeDeal()
		}()
	}
}

// SetReady is being called when we set ourselfs as ready.
func (g *Game) SetReady() {
	tablePos := g.playersList.getIndex(g.listenAddr)
	g.table.AddPlayerOnPosition(g.listenAddr, tablePos)

	g.playersReady.addRecvStatus(g.listenAddr)
	g.sendToPlayers(MessageReady{}, g.getOtherPlayers()...)
	g.setStatus(GameStatusPlayerReady)
}

func (g *Game) sendToPlayers(payload any, addr ...string) {
	g.broadcastch <- BroadcastTo{
		To:      addr,
		Payload: payload,
	}
}

func (g *Game) RemovePlayer(addr string) {
	idx := g.playersList.getIndex(addr)
	if idx == -1 {
		return
	}

	wasCurrentTurn := g.canTakeAction(addr)
	wasDealer := g.isFromCurrentDealer(addr)

	g.playersList.remove(addr)
	g.table.RemovePlayerByAddr(addr)

	logrus.WithFields(logrus.Fields{
		"addr": addr,
	}).Info("player removed from game state")

	if g.playersList.len() < 2 {
		logrus.Info("not enough players to continue, resetting game state")
		g.setStatus(GameStatusConnected)
		return
	}

	currTurn := int(g.currentPlayerTurn.Get())
	if idx < currTurn {
		g.currentPlayerTurn.Set(int32(currTurn - 1))
	} else if idx == currTurn {
		if currTurn >= g.playersList.len() {
			g.currentPlayerTurn.Set(0)
		}
	}

	dealerIdx := int(g.currentDealer.Get())
	if idx < dealerIdx {
		g.currentDealer.Set(int32(dealerIdx - 1))
	} else if idx == dealerIdx {
		if dealerIdx >= g.playersList.len() {
			g.currentDealer.Set(0)
		}
	}

	if GameStatus(g.currentStatus.Get()) != GameStatusConnected && GameStatus(g.currentStatus.Get()) != GameStatusPlayerReady {
		if wasCurrentTurn {
			logrus.Info("disconnected player was the current turn; skipping to next turn")
			if wasDealer {
				g.advanceToNexRound()
			}
		}
	}
}

func (g *Game) AddPlayer(from string) {
	// If the player is being added to the game. We are going to assume
	// that he is ready to play.
	g.playersList.add(from)
	sort.Sort(g.playersList)
}

func (g *Game) loop() {
	ticker := time.NewTicker(time.Second * 5)

	for {
		<-ticker.C

		currentDealerAddr, _ := g.getCurrentDealerAddr()
		logrus.WithFields(logrus.Fields{
			"we":             g.listenAddr,
			"playersList":    g.playersList.List(),
			"gameState":      GameStatus(g.currentStatus.Get()),
			"currentDealer":  currentDealerAddr,
			"nextPlayerTurn": g.currentPlayerTurn,
		}).Info()
	}
}

func (g *Game) getOtherPlayers() []string {
	players := []string{}

	for _, addr := range g.playersList.List() {
		if addr == g.listenAddr {
			continue
		}
		players = append(players, addr)
	}

	return players
}

// getPositionOnTable return the index of our own position on the table.
func (g *Game) getPositionOnTable() int {
	for i := 0; i < g.playersList.len(); i++ {
		if g.playersList.get(i) == g.listenAddr {
			return i
		}
	}

	panic("player does not exist in the playersList; that should not happen!!!")
}

// func (g *Game) getPrevPositionOnTable() int {
// 	ourPosition := g.getPositionOnTable()

// 	// if we are the in the first position on the table we need to return the last
// 	// index of the PlayersList.
// 	if ourPosition == 0 {
// 		return g.playersList.len() - 1
// 	}

// 	return ourPosition - 1
// }

func (g *Game) getNextDealer() int {
	current := int(g.currentDealer.Get())
	if g.playersList.len() == 0 {
		return 0
	}
	return (current + 1) % g.playersList.len()
}

// getNextPositionOnTable returns the index of the next player in the PlayersList.
// func (g *Game) getNextPositionOnTable() int {
// 	ourPosition := g.getPositionOnTable()

// 	// check if we are on the last position of the table, if so return first index 0.
// 	if ourPosition == g.playersList.len()-1 {
// 		return 0
// 	}

// 	return ourPosition + 1
// }
