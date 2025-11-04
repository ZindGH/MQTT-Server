package store

// Store defines the interface for persistent storage
type Store interface {
	// Session management
	SaveSession(clientID string, session *Session) error
	LoadSession(clientID string) (*Session, error)
	DeleteSession(clientID string) error

	// Message queue operations
	EnqueueMessage(clientID string, msg *Message) error
	DequeueMessages(clientID string) ([]*Message, error)

	// Retained messages
	StoreRetained(topic string, msg *Message) error
	GetRetained(topic string) (*Message, error)

	// QoS state tracking
	PersistInflight(clientID string, packetID uint16, msg *Message) error
	ClearInflight(clientID string, packetID uint16) error

	// Close the store
	Close() error
}

// Session represents a client session
type Session struct {
	ClientID      string
	CleanSession  bool
	Subscriptions []Subscription
}

// Subscription represents a topic subscription
type Subscription struct {
	Topic string
	QoS   byte
}

// Message represents an MQTT message
type Message struct {
	Topic   string
	Payload []byte
	QoS     byte
	Retain  bool
}
