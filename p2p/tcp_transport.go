package p2p

import (
	"encoding/gob"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

type NetAddr string

func (n NetAddr) String() string  { return string(n) }
func (n NetAddr) Network() string { return "tcp" }

type Peer struct {
	conn       net.Conn
	outbound   bool
	listenAddr string
	mu         sync.Mutex // protects concurrent writes to conn
	enc        *gob.Encoder
	dec        *gob.Decoder
}

func NewPeer(conn net.Conn, outbound bool) *Peer {
	return &Peer{
		conn:     conn,
		outbound: outbound,
		enc:      gob.NewEncoder(conn),
		dec:      gob.NewDecoder(conn),
	}
}

func (p *Peer) Send(payload any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enc.Encode(payload)
}

func (p *Peer) ReadLoop(msgch chan *Message, delch chan *Peer) {
	for {
		msg := new(Message)
		if err := p.dec.Decode(msg); err != nil {
			logrus.Errorf("decode message error from %s: %s", p.conn.RemoteAddr(), err)
			break
		}

		msgch <- msg
	}

	p.conn.Close()
	delch <- p
}

type TCPTransport struct {
	listenAddr string
	listener   net.Listener
	AddPeer    chan *Peer
	DelPeer    chan *Peer
}

func NewTCPTransport(addr string) *TCPTransport {
	return &TCPTransport{
		listenAddr: addr,
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		return err
	}

	t.listener = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			logrus.Error(err)
			continue
		}

		peer := NewPeer(conn, false)

		t.AddPeer <- peer
	}
}
