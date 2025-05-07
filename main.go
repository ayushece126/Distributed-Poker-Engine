package main

import (
	"time"

	"github.com/ayushece126/poker/p2p"
)

func makeServerAndStart(addr string) *p2p.Server {
	cfg := p2p.ServerConfig{
		Version:     "GGPOKER V0.1-alpha",
		ListenAddr:  addr,
		GameVariant: p2p.TexasHoldem,
	}
	server := p2p.NewServer(cfg)
	go server.Start()

	time.Sleep(200 * time.Millisecond)

	return server
}

func main() {
	playerA := makeServerAndStart(":3000")
	playerB := makeServerAndStart(":4000")
	playerC := makeServerAndStart(":8080")
	playerD := makeServerAndStart(":6000")
	playerE := makeServerAndStart(":8089")
	playerF := makeServerAndStart(":5080")

	time.Sleep(200 * time.Millisecond)
	playerB.Connect(playerA.ListenAddr)
	time.Sleep(200 * time.Millisecond)
	playerC.Connect(playerB.ListenAddr)
	time.Sleep(200 * time.Millisecond)
	playerD.Connect(playerC.ListenAddr)
	time.Sleep(200 * time.Millisecond)
	playerE.Connect(playerD.ListenAddr)
	time.Sleep(200 * time.Millisecond)
	playerF.Connect(playerE.ListenAddr)

	select {}
}
