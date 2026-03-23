package main

import (
	"net/http"
	"time"

	"github.com/anthdm/ggpoker/p2p"
)

func makeServerAndStart(addr, apiAddr string) *p2p.Server {
	cfg := p2p.ServerConfig{
		Version:       "GGPOKER V0.2-alpha",
		ListenAddr:    addr,
		APIListenAddr: apiAddr,
		GameVariant:   p2p.TexasHoldem,
	}
	server := p2p.NewServer(cfg)
	go server.Start()

	time.Sleep(time.Millisecond * 200)

	return server
}

func main() {
	playerA := makeServerAndStart(":13000", ":13001") // dealer
	playerB := makeServerAndStart(":14000", ":14001") // sb
	playerC := makeServerAndStart(":15000", ":15001") // bb
	playerD := makeServerAndStart(":16000", ":16001") // bb + 2

	go func() {
		time.Sleep(time.Second * 2)
		http.Get("http://localhost:13001/ready")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:14001/ready")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:15001/ready")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:16001/ready")

		// Wait for dealer's 5s timer + the initial encrypt/decrypt hole cards ring
		// This takes some time over the network
		time.Sleep(time.Second * 12)

		// --- PREFLOP ---
		// Everyone checks
		http.Get("http://localhost:14001/check")
		time.Sleep(time.Second * 1)
		http.Get("http://localhost:15001/check")
		time.Sleep(time.Second * 1)
		http.Get("http://localhost:16001/check")
		time.Sleep(time.Second * 1)
		http.Get("http://localhost:13001/check")

		// Wait for the Flop 3 community cards to be unspooled mathematically
		time.Sleep(time.Second * 8)

		// --- FLOP ---
		// 14000 folds
		// 15000 bets 100
		// 16000 calls the 100
		// 13000 calls the 100
		http.Get("http://localhost:14001/fold")
		time.Sleep(time.Second * 1)
		http.Get("http://localhost:15001/bet/100")
		time.Sleep(time.Second * 1)
		http.Get("http://localhost:16001/call")
		time.Sleep(time.Second * 1)
		http.Get("http://localhost:13001/call")

		// Wait for the Turn 1 community card to be unspooled
		time.Sleep(time.Second * 5)

		// --- TURN ---
		// 14000 is folded.
		// 15000 checks
		// 16000 bets 200
		// 13000 calls 200
		// 15000 (who checked earlier) now folds to the bet
		http.Get("http://localhost:15001/check")
		time.Sleep(time.Second * 1)
		http.Get("http://localhost:16001/bet/200")
		time.Sleep(time.Second * 1)
		http.Get("http://localhost:13001/call")
		time.Sleep(time.Second * 1)
		http.Get("http://localhost:15001/fold")

		// Wait for the River 1 community card to be unspooled
		time.Sleep(time.Second * 5)

		// --- RIVER ---
		// Only 16000 and 13000 are left.
		// Both check, leading to a Showdown!
		http.Get("http://localhost:16001/check")
		time.Sleep(time.Second * 1)
		http.Get("http://localhost:13001/check")

		// The game will automatically transition to showdown, calculate the hands, and award the pot!
	}()

	time.Sleep(time.Millisecond * 200)
	playerB.Connect(playerA.ListenAddr)

	time.Sleep(time.Millisecond * 200)
	playerC.Connect(playerB.ListenAddr)

	time.Sleep(time.Millisecond * 200)
	playerD.Connect(playerC.ListenAddr)

	select {}
}
