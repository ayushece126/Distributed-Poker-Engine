package p2p

import (
	"testing"
	"time"
)

func makeTestServer(addr, apiAddr string) *Server {
	cfg := ServerConfig{
		Version:       "GGPOKER V0.2-alpha",
		ListenAddr:    addr,
		APIListenAddr: apiAddr,
		GameVariant:   TexasHoldem,
	}
	server := NewServer(cfg)
	go server.Start()
	time.Sleep(time.Millisecond * 200)
	return server
}

func TestPeerDisconnection(t *testing.T) {
	nodeA := makeTestServer(":3010", ":3011")
	nodeB := makeTestServer(":4010", ":4011")
	nodeC := makeTestServer(":5010", ":5011")

	nodeB.Connect(nodeA.ListenAddr)
	time.Sleep(time.Millisecond * 400)
	nodeC.Connect(nodeB.ListenAddr)
	time.Sleep(time.Millisecond * 400)

	// Since they mesh and share peer lists, eventually everyone should have 3 players
	if nodeA.gameState.playersList.len() != 3 {
		t.Errorf("nodeA expected 3 players, got %d", nodeA.gameState.playersList.len())
	}

	// Simulate nodeB crashing by closing its active peer connections
	for _, p := range nodeB.peers {
		p.conn.Close()
	}
	// Give time for TCP connection dropping and goroutines to process delPeer channel
	time.Sleep(time.Millisecond * 400)

	if nodeA.gameState.playersList.len() != 2 {
		t.Errorf("nodeA expected 2 players after drop, got %d", nodeA.gameState.playersList.len())
	}
	if nodeC.gameState.playersList.len() != 2 {
		t.Errorf("nodeC expected 2 players after drop, got %d", nodeC.gameState.playersList.len())
	}
}
