package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	redisClient "redis-go/client"
	"time"
)

const defaultListenAddr = ":3333"

type Config struct {
	ListenAddr string
}

type Message struct {
	data []byte
	peer *Peer
}

type Server struct {
	Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	quitCh    chan struct{}
	msgCh     chan Message

	//
	kv *KV
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
		msgCh:     make(chan Message),
		kv:        NewKV(),
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

func (s *Server) handleMessage(msg Message) error {
	cmd, err := parseCommand(string(msg.data))
	if err != nil {
		return err
	}
	switch c := cmd.(type) {
	case SetCommand:
		//fmt.Println("got command to set:", c.key, c.value)
		return s.kv.Set(c.key, c.value)
	case GetCommand:
		//fmt.Println("got command to get:", c.key)
		val, ok := s.kv.Get(c.key)
		if !ok {
			return fmt.Errorf("key %s not found", c.key)
		}
		_, err = msg.peer.Send(val)
		if err != nil {
			slog.Error("peer send Error", "err", err)
			return err
		}
	}

	return nil
}

func (s *Server) loop() {
	for {
		select {
		// Add a new peer.
		case msg := <-s.msgCh:
			if err := s.handleMessage(msg); err != nil {
				slog.Error("raw message error", "err", err)
			}
		case peer := <-s.addPeerCh:
			// Add the peer to the peers map.
			s.peers[peer] = true
		case <-s.quitCh:
			// Close all peers.
			return
			//default:
			//	fmt.Println("default")
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
	peer := NewPeer(conn, s.msgCh)
	s.addPeerCh <- peer
	//slog.Info("new peer connected", "remoteAddr", conn.RemoteAddr())
	if err := peer.readLoop(); err != nil {
		slog.Error("peer read error", err, "remoteAddr", conn.RemoteAddr())
	}
}

func main() {
	server := NewServer(Config{})
	go func() {
		log.Fatal(server.Start())
	}()
	time.Sleep(time.Second)

	client := redisClient.New("localhost:3333")
	for i := 0; i < 10; i++ {
		if err := client.Set(context.Background(), fmt.Sprintf("foo_%d", i), fmt.Sprintf("bar_%d", i)); err != nil {
			log.Fatal(err)
		}
		time.Sleep(10 * time.Microsecond) // conn.dial for set might be slower than conn.dial for get
		val, err := client.Get(context.Background(), fmt.Sprintf("foo_%d", i))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("got this back:", val)
	}
	fmt.Println(server.kv.data)
	time.Sleep(time.Second) // We are blocking here so the program doesnt exits
}
