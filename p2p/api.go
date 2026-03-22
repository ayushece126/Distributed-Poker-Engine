package p2p

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type MyError struct {
	err error
}

func (e MyError) Error() string {
	return e.err.Error()
}

type apiFunc func(w http.ResponseWriter, r *http.Request) error

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
	}
}

func JSON(w http.ResponseWriter, status int, v any) error {
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type APIServer struct {
	listenAddr string
	server     *Server
}

func NewAPIServer(listenAddr string, server *Server) *APIServer {
	return &APIServer{
		server:     server,
		listenAddr: listenAddr,
	}
}

func (s *APIServer) Run() {
	r := mux.NewRouter()

	r.HandleFunc("/ready", makeHTTPHandleFunc(s.handlePlayerReady))
	r.HandleFunc("/fold", makeHTTPHandleFunc(s.handlePlayerFold))
	r.HandleFunc("/check", makeHTTPHandleFunc(s.handlePlayerCheck))
	r.HandleFunc("/call", makeHTTPHandleFunc(s.handlePlayerCall))
	r.HandleFunc("/bet/{value}", makeHTTPHandleFunc(s.handlePlayerBet))
	r.HandleFunc("/raise/{value}", makeHTTPHandleFunc(s.handlePlayerRaise))

	http.ListenAndServe(s.listenAddr, r)
}

// sendAction routes a player action through the server's serialized channel
// and blocks until the server loop processes it.
func (s *APIServer) sendAction(action PlayerAction, value int) error {
	result := make(chan error, 1)
	s.server.apiActionCh <- APIAction{
		Action: action,
		Value:  value,
		Result: result,
	}
	return <-result
}

func (s *APIServer) handlePlayerBet(w http.ResponseWriter, r *http.Request) error {
	valueStr := mux.Vars(r)["value"]
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return err
	}

	if err := s.sendAction(PlayerActionBet, value); err != nil {
		return err
	}

	return JSON(w, http.StatusOK, fmt.Sprintf("value:%d", value))
}

func (s *APIServer) handlePlayerCheck(w http.ResponseWriter, r *http.Request) error {
	if err := s.sendAction(PlayerActionCheck, 0); err != nil {
		return err
	}
	return JSON(w, http.StatusOK, "CHECKED")
}

func (s *APIServer) handlePlayerFold(w http.ResponseWriter, r *http.Request) error {
	if err := s.sendAction(PlayerActionFold, 0); err != nil {
		return err
	}
	return JSON(w, http.StatusOK, "FOLDED")
}

func (s *APIServer) handlePlayerReady(w http.ResponseWriter, r *http.Request) error {
	result := make(chan error, 1)
	s.server.apiReadyCh <- APIReady{Result: result}
	if err := <-result; err != nil {
		return err
	}
	return JSON(w, http.StatusOK, "READY")
}

func (s *APIServer) handlePlayerCall(w http.ResponseWriter, r *http.Request) error {
	if err := s.sendAction(PlayerActionCall, 0); err != nil {
		return err
	}
	return JSON(w, http.StatusOK, "CALLED")
}

func (s *APIServer) handlePlayerRaise(w http.ResponseWriter, r *http.Request) error {
	valueStr := mux.Vars(r)["value"]
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return err
	}
	if err := s.sendAction(PlayerActionRaise, value); err != nil {
		return err
	}
	return JSON(w, http.StatusOK, fmt.Sprintf("RAISED to %d", value))
}
