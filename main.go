package main

import (
	"log"
	"log/slog"
	"net"
)

const defaultListenAddr = ":3333"

type Config struct {
	ListenAddr string
}

type Server struct {
	Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	quitCh    chan struct{}
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddr) == 0 {
		cfg.ListenAddr = defaultListenAddr
	}
	return &Server{
		Config:    cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quitCh:    make(chan struct{}),
	}
}

func (s *Server) Start() error {
	/*
		1. Listen for incoming connections
		2. Accept incoming connections
		3. Create a new peer
		4. Add the peer to the peers map
		5. Start the read loop for the peer
		6. Handle the connection
		7. Close all peers
		8. Close the server
	*/
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	s.ln = ln
	go s.loop()
	slog.Info("server started", "addr", s.ListenAddr)
	return s.acceptLoop()
}

func (s *Server) loop() {
	for {
		select {
		// Add a new peer.
		case peer := <-s.addPeerCh:
			// Add the peer to the peers map.
			s.peers[peer] = true
		case <-s.quitCh:
			// Close all peers.
			return
		}
	}
}

func (s *Server) acceptLoop() error {
	for {
		// Accept a new connection.
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("accept error", err)
			continue
		}
		go s.handleConn(conn)
	}
}
func (s *Server) handleConn(conn net.Conn) {
	// Create a new peer.
	peer := NewPeer(conn)
	s.addPeerCh <- peer
	slog.Info("new peer connected", "addr", conn.RemoteAddr())
	peer.readLoop()
}

func main() {
	server := NewServer(Config{})
	log.Fatal(server.Start())
}
