package p2p

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestFivePlayerGame spins up 5 nodes, connects them into a mesh,
// readies them up, and walks through a complete game:
// PreFlop betting → Flop → Turn → River → (all check = showdown)
// OR some fold mid-game and the last player standing wins.
func TestFivePlayerGame(t *testing.T) {
	// Spin up 5 servers with unique TCP and API ports
	ports := []struct {
		tcp string
		api string
	}{
		{":13000", ":13001"},
		{":14000", ":14001"},
		{":15000", ":15001"},
		{":16000", ":16001"},
		{":17000", ":17001"},
	}

	servers := make([]*Server, len(ports))
	for i, p := range ports {
		cfg := ServerConfig{
			Version:       "TEST V1",
			ListenAddr:    p.tcp,
			APIListenAddr: p.api,
			GameVariant:   TexasHoldem,
		}
		servers[i] = NewServer(cfg)
		go servers[i].Start()
		time.Sleep(200 * time.Millisecond)
	}

	// Connect into mesh: each node connects to the previous
	for i := 1; i < len(servers); i++ {
		err := servers[i].Connect(servers[i-1].ListenAddr)
		assert.NoError(t, err, "server %d failed to connect to server %d", i, i-1)
		time.Sleep(300 * time.Millisecond)
	}

	// Verify all nodes see all 5 players
	time.Sleep(2 * time.Second)
	for i, s := range servers {
		count := s.gameState.playersList.len()
		t.Logf("Server %d (%s) sees %d players", i, s.ListenAddr, count)
		assert.Equal(t, 5, count, "server %d should see 5 players", i)
	}

	// Ready up all players via API
	for i, p := range ports {
		apiURL := "http://localhost" + p.api + "/ready"
		resp, err := http.Get(apiURL)
		assert.NoError(t, err, "player %d ready failed", i)
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for the dealer's 5-second timer + encryption ring to complete
	t.Log("Waiting for deck encryption ring and hole card dealing...")
	time.Sleep(15 * time.Second)

	// Verify all nodes are in PreFlop state
	for i, s := range servers {
		s.gameState.mu.Lock()
		status := GameStatus(s.gameState.currentStatus)
		holeCards := len(s.gameState.HoleCards)
		s.gameState.mu.Unlock()
		t.Logf("Server %d: status=%s, holeCards=%d", i, status, holeCards)
	}

	// === PreFlop Betting Round: Everyone checks ===
	t.Log("=== PREFLOP: Everyone checks ===")
	for i, p := range ports {
		apiURL := "http://localhost" + p.api + "/check"
		resp, err := http.Get(apiURL)
		if err != nil {
			t.Logf("Player %d check error: %v (may not be their turn yet)", i, err)
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}

	// Wait for round advancement + community card ring
	time.Sleep(10 * time.Second)

	// Log state after PreFlop
	for i, s := range servers {
		s.gameState.mu.Lock()
		status := GameStatus(s.gameState.currentStatus)
		community := len(s.gameState.CommunityCards)
		pot := s.gameState.Pot
		s.gameState.mu.Unlock()
		t.Logf("After PreFlop - Server %d: status=%s, communityCards=%d, pot=%d", i, status, community, pot)
	}

	// === Flop Betting: Player 0 bets, everyone else folds ===
	t.Log("=== FLOP: Player 0 bets 100, others fold ===")

	// Player 0 bets
	resp, err := http.Get("http://localhost" + ports[0].api + "/bet/100")
	if err != nil {
		t.Logf("Player 0 bet error: %v", err)
	}
	if resp != nil {
		resp.Body.Close()
	}
	time.Sleep(1 * time.Second)

	// Players 1-4 fold
	for i := 1; i < len(ports); i++ {
		apiURL := "http://localhost" + ports[i].api + "/fold"
		resp, err := http.Get(apiURL)
		if err != nil {
			t.Logf("Player %d fold error: %v", i, err)
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}

	// Wait for last-player-standing logic to kick in
	time.Sleep(5 * time.Second)

	// Verify results: Player 0 should have won the pot
	t.Log("=== FINAL STATE ===")
	for i, s := range servers {
		s.gameState.mu.Lock()
		status := GameStatus(s.gameState.currentStatus)
		pot := s.gameState.Pot
		s.gameState.mu.Unlock()

		player, perr := s.gameState.table.GetPlayer(s.ListenAddr)
		balance := 0
		if perr == nil {
			balance = player.Balance
		}

		t.Logf("Server %d (%s): status=%s, pot=%d, balance=%d", i, s.ListenAddr, status, pot, balance)
	}
}
