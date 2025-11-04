package server

import (
	"fmt"
	"net"
	"sync"
)

// Server represents the MQTT broker server
type Server struct {
	listener net.Listener
	port     int
	mu       sync.Mutex
	running  bool
	clients  map[string]*Client // clientID -> Client
}

// Client represents a connected MQTT client
type Client struct {
	ID   string
	Conn net.Conn
}

// New creates a new MQTT server instance
func New() (*Server, error) {
	return &Server{
		port:    1883,
		clients: make(map[string]*Client),
	}, nil
}

// Start begins listening for MQTT connections
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.running = true
	s.mu.Unlock()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	s.listener = listener

	fmt.Printf("MQTT broker listening on port %d\n", s.port)

	// Accept connections
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			running := s.running
			s.mu.Unlock()
			if !running {
				return nil // Server stopped
			}
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		// Handle each connection in a goroutine
		go s.handleConnection(conn)
	}
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	// Close listener
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return fmt.Errorf("error closing listener: %w", err)
		}
	}

	// Close all client connections
	for _, client := range s.clients {
		if client.Conn != nil {
			client.Conn.Close()
		}
	}

	return nil
}

// handleConnection processes an individual client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Printf("New connection from %s\n", conn.RemoteAddr())

	// TODO: Implement MQTT protocol handling
	// For now, just read and echo
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("Connection closed: %v\n", err)
			return
		}

		fmt.Printf("Received %d bytes from %s\n", n, conn.RemoteAddr())
		// TODO: Parse MQTT packets and handle them
	}
}
