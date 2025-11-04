package store

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"go.etcd.io/bbolt"
)

var (
	// Bucket names
	sessionsBucket = []byte("sessions")
	messagesBucket = []byte("messages")
	retainedBucket = []byte("retained")
	inflightBucket = []byte("inflight")
)

// BboltStore implements Store interface using bbolt embedded database
type BboltStore struct {
	db *bbolt.DB
}

// NewBboltStore creates a new bbolt-backed store
func NewBboltStore(path string) (*BboltStore, error) {
	// Note: bbolt will create the file if it doesn't exist
	// The directory must exist beforehand
	_ = filepath.Dir(path) // Directory validation could be added here

	// Open database
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bbolt.Tx) error {
		buckets := [][]byte{sessionsBucket, messagesBucket, retainedBucket, inflightBucket}
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &BboltStore{db: db}, nil
}

// SaveSession stores a client session
func (s *BboltStore) SaveSession(clientID string, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(sessionsBucket)
		return bucket.Put([]byte(clientID), data)
	})
}

// LoadSession retrieves a client session
func (s *BboltStore) LoadSession(clientID string) (*Session, error) {
	var session Session

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(sessionsBucket)
		data := bucket.Get([]byte(clientID))
		if data == nil {
			return fmt.Errorf("session not found")
		}
		return json.Unmarshal(data, &session)
	})

	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteSession removes a client session
func (s *BboltStore) DeleteSession(clientID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(sessionsBucket)
		return bucket.Delete([]byte(clientID))
	})
}

// EnqueueMessage adds a message to a client's queue
func (s *BboltStore) EnqueueMessage(clientID string, msg *Message) error {
	// Create a queue key: clientID + message timestamp/ID
	queueKey := fmt.Sprintf("%s:%d", clientID, len(msg.Payload)) // Simplified for now

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(messagesBucket)
		return bucket.Put([]byte(queueKey), data)
	})
}

// DequeueMessages retrieves all queued messages for a client
func (s *BboltStore) DequeueMessages(clientID string) ([]*Message, error) {
	var messages []*Message

	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(messagesBucket)
		cursor := bucket.Cursor()

		// Find all messages for this client
		prefix := []byte(clientID + ":")
		for k, v := cursor.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = cursor.Next() {
			var msg Message
			if err := json.Unmarshal(v, &msg); err != nil {
				return err
			}
			messages = append(messages, &msg)

			// Delete the message after reading
			if err := cursor.Delete(); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return messages, nil
}

// StoreRetained stores a retained message for a topic
func (s *BboltStore) StoreRetained(topic string, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal retained message: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(retainedBucket)
		return bucket.Put([]byte(topic), data)
	})
}

// GetRetained retrieves the retained message for a topic
func (s *BboltStore) GetRetained(topic string) (*Message, error) {
	var msg Message

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(retainedBucket)
		data := bucket.Get([]byte(topic))
		if data == nil {
			return fmt.Errorf("no retained message for topic")
		}
		return json.Unmarshal(data, &msg)
	})

	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// PersistInflight stores an in-flight QoS 1/2 message
func (s *BboltStore) PersistInflight(clientID string, packetID uint16, msg *Message) error {
	key := fmt.Sprintf("%s:%d", clientID, packetID)

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal inflight message: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(inflightBucket)
		return bucket.Put([]byte(key), data)
	})
}

// ClearInflight removes an in-flight message after acknowledgment
func (s *BboltStore) ClearInflight(clientID string, packetID uint16) error {
	key := fmt.Sprintf("%s:%d", clientID, packetID)

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(inflightBucket)
		return bucket.Delete([]byte(key))
	})
}

// Close closes the database
func (s *BboltStore) Close() error {
	return s.db.Close()
}

// Stats returns database statistics
func (s *BboltStore) Stats() map[string]interface{} {
	stats := s.db.Stats()
	return map[string]interface{}{
		"tx_stats": stats.TxStats,
	}
}
