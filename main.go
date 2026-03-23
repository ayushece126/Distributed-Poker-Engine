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

		// // [3000:D, 4000:sb, 6000:bb, 7000]
		// PREFLOP
		time.Sleep(time.Second * 10)

		http.Get("http://localhost:14001/check")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:15001/check")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:16001/check")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:13001/check")

		// FLOP
		time.Sleep(time.Second * 5)
		http.Get("http://localhost:14001/fold")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:15001/bet/100")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:16001/call")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:13001/call")

		// // TURN
		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:4001/fold")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:6001/fold")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:7001/fold")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:3001/fold")

		// // RIVER
		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:4001/fold")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:6001/fold")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:7001/fold")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:3001/fold")

	}()

	time.Sleep(time.Millisecond * 200)
	playerB.Connect(playerA.ListenAddr)

	time.Sleep(time.Millisecond)
	playerC.Connect(playerB.ListenAddr)

	time.Sleep(time.Millisecond * 200)
	playerD.Connect(playerC.ListenAddr)

	select {}
}
