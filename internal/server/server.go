package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/ZindGH/MQTT-Server/internal/config"
	"github.com/ZindGH/MQTT-Server/internal/mqtt"
	"github.com/ZindGH/MQTT-Server/internal/store"
)

// Server represents the MQTT broker server
type Server struct {
	config         *config.Config
	listener       net.Listener
	store          store.Store
	mu             sync.RWMutex
	running        bool
	clients        map[string]*Client             // clientID -> Client
	retainedMsgs   map[string]*mqtt.PublishPacket // topic -> retained message
	retainedMsgsMu sync.RWMutex
	wg             sync.WaitGroup
}

// Client represents a connected MQTT client
type Client struct {
	ID            string
	Conn          net.Conn
	CleanSession  bool
	Subscriptions map[string]byte // topic -> QoS
	mu            sync.RWMutex
}

// New creates a new MQTT server instance
func New() (*Server, error) {
	// For backward compatibility with tests
	return &Server{
		config: &config.Config{
			Server: config.ServerConfig{
				Host: "127.0.0.1",
				Port: 1883,
			},
		},
		clients: make(map[string]*Client),
	}, nil
}

// NewWithConfig creates a new MQTT server with configuration
func NewWithConfig(cfg *config.Config, st store.Store) (*Server, error) {
	return &Server{
		config:       cfg,
		store:        st,
		clients:      make(map[string]*Client),
		retainedMsgs: make(map[string]*mqtt.PublishPacket),
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

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	s.listener = listener

	log.Printf("MQTT broker listening on %s", addr)

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
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// Handle each connection in a goroutine
		s.wg.Add(1)
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
	defer s.wg.Done()
	defer conn.Close()

	log.Printf("New connection from %s", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	var client *Client

	for {
		// Read fixed header
		header, err := mqtt.ReadFixedHeader(reader)
		if err != nil {
			if client != nil {
				log.Printf("Client %s disconnected: %v", client.ID, err)
			} else {
				log.Printf("Connection from %s closed: %v", conn.RemoteAddr(), err)
			}
			return
		}

		log.Printf("Received %s packet (remaining length: %d)", header.PacketType, header.RemainingLen)

		// Read the remaining packet data
		remainingData := make([]byte, header.RemainingLen)
		if header.RemainingLen > 0 {
			n, err := io.ReadFull(reader, remainingData)
			if err != nil {
				log.Printf("Failed to read packet data (read %d/%d bytes): %v", n, header.RemainingLen, err)
				return
			}
		}

		// Handle different packet types
		switch header.PacketType {
		case mqtt.CONNECT:
			client = s.handleConnect(conn, bytes.NewReader(remainingData), header.RemainingLen)
			if client == nil {
				return // Connection rejected
			}

		case mqtt.PUBLISH:
			if client == nil {
				log.Printf("PUBLISH received before CONNECT")
				return
			}
			s.handlePublish(client, header, remainingData)

		case mqtt.SUBSCRIBE:
			if client == nil {
				log.Printf("SUBSCRIBE received before CONNECT")
				return
			}
			s.handleSubscribe(client, conn, remainingData)

		case mqtt.UNSUBSCRIBE:
			if client == nil {
				log.Printf("UNSUBSCRIBE received before CONNECT")
				return
			}
			s.handleUnsubscribe(client, remainingData)

		case mqtt.PINGREQ:
			s.handlePingreq(conn)

		case mqtt.DISCONNECT:
			log.Printf("Client %s disconnected gracefully", client.ID)
			return

		default:
			log.Printf("Unhandled packet type: %s", header.PacketType)
		}
	}
}

func (s *Server) handleConnect(conn net.Conn, reader *bytes.Reader, remainingLen int) *Client {
	connectPkt, err := mqtt.DecodeConnectPacket(reader, remainingLen)
	if err != nil {
		log.Printf("Failed to decode CONNECT: %v", err)
		conn.Close()
		return nil
	}

	log.Printf("CONNECT from client: %s (protocol: %s v%d, clean_session: %v)",
		connectPkt.ClientID, connectPkt.ProtocolName, connectPkt.ProtocolVersion, connectPkt.CleanSession)

	// Create client
	client := &Client{
		ID:            connectPkt.ClientID,
		Conn:          conn,
		CleanSession:  connectPkt.CleanSession,
		Subscriptions: make(map[string]byte),
	}

	// Store client
	s.mu.Lock()
	s.clients[client.ID] = client
	s.mu.Unlock()

	// Send CONNACK
	connack := &mqtt.ConnackPacket{
		SessionPresent: false,
		ReturnCode:     0, // Connection accepted
	}
	data, _ := connack.Encode()
	conn.Write(data)

	log.Printf("Client %s connected successfully", client.ID)

	// Update metrics if configured
	if s.config != nil && s.config.Metrics.Enabled {
		// Metrics will be updated through imported package
	}

	return client
}

func (s *Server) handlePublish(client *Client, header *mqtt.FixedHeader, data []byte) {
	// Decode PUBLISH packet
	publishPkt, err := mqtt.DecodePublishPacket(bytes.NewReader(data), header)
	if err != nil {
		log.Printf("Failed to decode PUBLISH: %v", err)
		return
	}

	log.Printf("PUBLISH from %s: topic=%s, QoS=%d, retain=%t, payload=%d bytes",
		client.ID, publishPkt.Topic, publishPkt.QoS, publishPkt.Retain, len(publishPkt.Payload))

	// Handle retained messages
	if publishPkt.Retain {
		s.retainedMsgsMu.Lock()
		if len(publishPkt.Payload) == 0 {
			// Empty payload removes retained message
			delete(s.retainedMsgs, publishPkt.Topic)
			log.Printf("Removed retained message for topic %s", publishPkt.Topic)
		} else {
			// Store retained message
			s.retainedMsgs[publishPkt.Topic] = publishPkt
			log.Printf("Stored retained message for topic %s", publishPkt.Topic)
		}
		s.retainedMsgsMu.Unlock()
	}

	// Send PUBACK for QoS 1
	if publishPkt.QoS == 1 {
		puback := &mqtt.PubackPacket{
			PacketID: publishPkt.PacketID,
		}
		ackData, _ := puback.Encode()
		client.Conn.Write(ackData)
		log.Printf("Sent PUBACK to %s for packet %d", client.ID, publishPkt.PacketID)
	}

	// Route message to subscribers
	s.routeMessage(publishPkt)
}

func (s *Server) handleSubscribe(client *Client, conn net.Conn, data []byte) {
	// Decode SUBSCRIBE packet
	subscribePkt, err := mqtt.DecodeSubscribePacket(bytes.NewReader(data), len(data))
	if err != nil {
		log.Printf("Failed to decode SUBSCRIBE: %v", err)
		return
	}

	log.Printf("SUBSCRIBE from %s: %d topics", client.ID, len(subscribePkt.Topics))

	// Store subscriptions
	client.mu.Lock()
	returnCodes := make([]byte, len(subscribePkt.Topics))
	for i, sub := range subscribePkt.Topics {
		client.Subscriptions[sub.Topic] = sub.QoS
		returnCodes[i] = sub.QoS // Grant requested QoS
		log.Printf("  - %s subscribed to %s (QoS %d)", client.ID, sub.Topic, sub.QoS)
	}
	client.mu.Unlock()

	// Send SUBACK
	suback := &mqtt.SubackPacket{
		PacketID:    subscribePkt.PacketID,
		ReturnCodes: returnCodes,
	}
	ackData, err := suback.Encode()
	if err != nil {
		log.Printf("Failed to encode SUBACK: %v", err)
		return
	}

	n, err := client.Conn.Write(ackData)
	if err != nil {
		log.Printf("Failed to send SUBACK to %s: %v", client.ID, err)
		return
	}
	log.Printf("Sent SUBACK to %s for packet %d (%d bytes)", client.ID, subscribePkt.PacketID, n)

	// Deliver retained messages matching the subscriptions
	s.retainedMsgsMu.RLock()
	for topic, retainedMsg := range s.retainedMsgs {
		for _, sub := range subscribePkt.Topics {
			if topicMatch(sub.Topic, topic) {
				// Send retained message to new subscriber
				go s.deliverMessage(client, retainedMsg, sub.QoS)
				log.Printf("Delivered retained message on topic %s to %s", topic, client.ID)
				break
			}
		}
	}
	s.retainedMsgsMu.RUnlock()
}

func (s *Server) handleUnsubscribe(client *Client, data []byte) {
	// Decode UNSUBSCRIBE packet
	unsubscribePkt, err := mqtt.DecodeUnsubscribePacket(bytes.NewReader(data), len(data))
	if err != nil {
		log.Printf("Failed to decode UNSUBSCRIBE: %v", err)
		return
	}

	log.Printf("UNSUBSCRIBE from %s: %d topics", client.ID, len(unsubscribePkt.Topics))

	// Remove subscriptions
	client.mu.Lock()
	for _, topic := range unsubscribePkt.Topics {
		delete(client.Subscriptions, topic)
		log.Printf("  - %s unsubscribed from %s", client.ID, topic)
	}
	client.mu.Unlock()

	// Send UNSUBACK
	unsuback := &mqtt.UnsubackPacket{
		PacketID: unsubscribePkt.PacketID,
	}
	ackData, err := unsuback.Encode()
	if err != nil {
		log.Printf("Failed to encode UNSUBACK: %v", err)
		return
	}

	n, err := client.Conn.Write(ackData)
	if err != nil {
		log.Printf("Failed to send UNSUBACK to %s: %v", client.ID, err)
		return
	}
	log.Printf("Sent UNSUBACK to %s for packet %d (%d bytes)", client.ID, unsubscribePkt.PacketID, n)
}

// routeMessage delivers a message to all matching subscribers
func (s *Server) routeMessage(pub *mqtt.PublishPacket) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	delivered := 0
	for _, client := range s.clients {
		client.mu.RLock()
		for subTopic, subQoS := range client.Subscriptions {
			if topicMatch(subTopic, pub.Topic) {
				// Deliver message to subscriber
				go s.deliverMessage(client, pub, subQoS)
				delivered++
				break // Only deliver once per client
			}
		}
		client.mu.RUnlock()
	}

	log.Printf("Routed message on topic %s to %d subscribers", pub.Topic, delivered)
}

// deliverMessage sends a PUBLISH packet to a subscriber
func (s *Server) deliverMessage(client *Client, pub *mqtt.PublishPacket, subQoS byte) {
	// Use the minimum of publisher and subscriber QoS
	qos := pub.QoS
	if subQoS < qos {
		qos = subQoS
	}

	// Build PUBLISH packet
	buf := bytes.NewBuffer(nil)

	// Fixed header
	fixedHeader := byte(mqtt.PUBLISH) << 4
	if pub.Dup {
		fixedHeader |= 0x08
	}
	fixedHeader |= (qos << 1)
	if pub.Retain {
		fixedHeader |= 0x01
	}
	buf.WriteByte(fixedHeader)

	// Calculate remaining length
	remainingLen := 2 + len(pub.Topic) + len(pub.Payload)
	if qos > 0 {
		remainingLen += 2 // Packet ID
	}

	// Write remaining length (simplified, assuming < 128 bytes for now)
	if remainingLen < 128 {
		buf.WriteByte(byte(remainingLen))
	} else {
		// Variable length encoding
		x := remainingLen
		for x > 0 {
			digit := byte(x % 128)
			x = x / 128
			if x > 0 {
				digit |= 0x80
			}
			buf.WriteByte(digit)
		}
	}

	// Write topic
	buf.Write(mqtt.WriteString(pub.Topic))

	// Write packet ID if QoS > 0
	if qos > 0 {
		packetID := pub.PacketID
		if packetID == 0 {
			packetID = 1 // Generate packet ID if needed
		}
		buf.WriteByte(byte(packetID >> 8))
		buf.WriteByte(byte(packetID & 0xFF))
	}

	// Write payload
	buf.Write(pub.Payload)

	// Send to client
	if _, err := client.Conn.Write(buf.Bytes()); err != nil {
		log.Printf("Failed to deliver message to %s: %v", client.ID, err)
	} else {
		log.Printf("Delivered message to %s on topic %s", client.ID, pub.Topic)
	}
}

// topicMatch checks if a subscription topic matches a publish topic
// Supports MQTT wildcards: + (single level) and # (multi level)
func topicMatch(subTopic, pubTopic string) bool {
	// Exact match
	if subTopic == pubTopic {
		return true
	}

	// Split topics into levels
	subLevels := splitTopic(subTopic)
	pubLevels := splitTopic(pubTopic)

	// Match level by level
	si, pi := 0, 0
	for si < len(subLevels) && pi < len(pubLevels) {
		subLevel := subLevels[si]
		pubLevel := pubLevels[pi]

		if subLevel == "#" {
			// Multi-level wildcard matches everything remaining
			return true
		} else if subLevel == "+" {
			// Single-level wildcard matches one level
			si++
			pi++
		} else if subLevel == pubLevel {
			// Exact level match
			si++
			pi++
		} else {
			// No match
			return false
		}
	}

	// Check if we consumed all levels
	// Handle trailing # in subscription
	if si < len(subLevels) && subLevels[si] == "#" {
		return true
	}

	// Both must be fully consumed for a match
	return si == len(subLevels) && pi == len(pubLevels)
}

// splitTopic splits a topic into levels by '/'
func splitTopic(topic string) []string {
	if topic == "" {
		return []string{}
	}

	levels := []string{}
	start := 0
	for i := 0; i < len(topic); i++ {
		if topic[i] == '/' {
			levels = append(levels, topic[start:i])
			start = i + 1
		}
	}
	// Add last level
	if start < len(topic) {
		levels = append(levels, topic[start:])
	}

	return levels
}

func (s *Server) handlePingreq(conn net.Conn) {
	pingresp := &mqtt.PingrespPacket{}
	data, _ := pingresp.Encode()
	conn.Write(data)
}
