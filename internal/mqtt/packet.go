package mqtt

import (
	"encoding/binary"
	"fmt"
	"io"
)

// PacketType represents MQTT packet types
type PacketType byte

const (
	CONNECT     PacketType = 1
	CONNACK     PacketType = 2
	PUBLISH     PacketType = 3
	PUBACK      PacketType = 4
	PUBREC      PacketType = 5
	PUBREL      PacketType = 6
	PUBCOMP     PacketType = 7
	SUBSCRIBE   PacketType = 8
	SUBACK      PacketType = 9
	UNSUBSCRIBE PacketType = 10
	UNSUBACK    PacketType = 11
	PINGREQ     PacketType = 12
	PINGRESP    PacketType = 13
	DISCONNECT  PacketType = 14
)

// String returns the packet type name
func (pt PacketType) String() string {
	names := map[PacketType]string{
		CONNECT:     "CONNECT",
		CONNACK:     "CONNACK",
		PUBLISH:     "PUBLISH",
		PUBACK:      "PUBACK",
		PUBREC:      "PUBREC",
		PUBREL:      "PUBREL",
		PUBCOMP:     "PUBCOMP",
		SUBSCRIBE:   "SUBSCRIBE",
		SUBACK:      "SUBACK",
		UNSUBSCRIBE: "UNSUBSCRIBE",
		UNSUBACK:    "UNSUBACK",
		PINGREQ:     "PINGREQ",
		PINGRESP:    "PINGRESP",
		DISCONNECT:  "DISCONNECT",
	}
	if name, ok := names[pt]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", pt)
}

// FixedHeader represents the MQTT fixed header
type FixedHeader struct {
	PacketType   PacketType
	Flags        byte
	RemainingLen int
}

// Packet represents a generic MQTT packet
type Packet interface {
	Type() PacketType
	Encode() ([]byte, error)
}

// ConnectPacket represents a CONNECT packet
type ConnectPacket struct {
	ProtocolName    string
	ProtocolVersion byte
	CleanSession    bool
	WillFlag        bool
	WillQoS         byte
	WillRetain      bool
	PasswordFlag    bool
	UsernameFlag    bool
	KeepAlive       uint16
	ClientID        string
	WillTopic       string
	WillMessage     []byte
	Username        string
	Password        []byte
}

func (c *ConnectPacket) Type() PacketType { return CONNECT }

// ConnackPacket represents a CONNACK packet
type ConnackPacket struct {
	SessionPresent bool
	ReturnCode     byte
}

func (c *ConnackPacket) Type() PacketType { return CONNACK }

// Encode creates a CONNACK packet bytes
func (c *ConnackPacket) Encode() ([]byte, error) {
	buf := make([]byte, 4)
	buf[0] = byte(CONNACK) << 4 // Fixed header
	buf[1] = 2                  // Remaining length
	if c.SessionPresent {
		buf[2] = 1
	}
	buf[3] = c.ReturnCode
	return buf, nil
}

// PublishPacket represents a PUBLISH packet
type PublishPacket struct {
	Dup      bool
	QoS      byte
	Retain   bool
	Topic    string
	PacketID uint16
	Payload  []byte
}

func (p *PublishPacket) Type() PacketType { return PUBLISH }

func (p *PublishPacket) Encode() ([]byte, error) {
	// TODO: Implement full PUBLISH encoding
	return nil, fmt.Errorf("PUBLISH encoding not yet implemented")
}

// PubackPacket represents a PUBACK packet
type PubackPacket struct {
	PacketID uint16
}

func (p *PubackPacket) Type() PacketType { return PUBACK }

func (p *PubackPacket) Encode() ([]byte, error) {
	buf := make([]byte, 4)
	buf[0] = byte(PUBACK) << 4
	buf[1] = 2
	binary.BigEndian.PutUint16(buf[2:], p.PacketID)
	return buf, nil
}

// SubscribePacket represents a SUBSCRIBE packet
type SubscribePacket struct {
	PacketID uint16
	Topics   []Subscription
}

type Subscription struct {
	Topic string
	QoS   byte
}

func (s *SubscribePacket) Type() PacketType { return SUBSCRIBE }

// SubackPacket represents a SUBACK packet
type SubackPacket struct {
	PacketID    uint16
	ReturnCodes []byte
}

func (s *SubackPacket) Type() PacketType { return SUBACK }

func (s *SubackPacket) Encode() ([]byte, error) {
	buf := make([]byte, 2+2+len(s.ReturnCodes))
	buf[0] = byte(SUBACK) << 4
	buf[1] = byte(2 + len(s.ReturnCodes))
	binary.BigEndian.PutUint16(buf[2:], s.PacketID)
	copy(buf[4:], s.ReturnCodes)
	return buf, nil
}

// UnsubscribePacket represents an UNSUBSCRIBE packet
type UnsubscribePacket struct {
	PacketID uint16
	Topics   []string
}

func (u *UnsubscribePacket) Type() PacketType { return UNSUBSCRIBE }

// UnsubackPacket represents an UNSUBACK packet
type UnsubackPacket struct {
	PacketID uint16
}

func (u *UnsubackPacket) Type() PacketType { return UNSUBACK }

func (u *UnsubackPacket) Encode() ([]byte, error) {
	buf := make([]byte, 4)
	buf[0] = byte(UNSUBACK) << 4
	buf[1] = 2
	binary.BigEndian.PutUint16(buf[2:], u.PacketID)
	return buf, nil
}

// PingrespPacket represents a PINGRESP packet
type PingrespPacket struct{}

func (p *PingrespPacket) Type() PacketType { return PINGRESP }

func (p *PingrespPacket) Encode() ([]byte, error) {
	return []byte{byte(PINGRESP) << 4, 0}, nil
}

// ReadFixedHeader reads the fixed header from a reader
func ReadFixedHeader(r io.Reader) (*FixedHeader, error) {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}

	header := &FixedHeader{
		PacketType: PacketType((buf[0] >> 4) & 0x0F),
		Flags:      buf[0] & 0x0F,
	}

	// Read remaining length (variable length encoding)
	multiplier := 1
	value := 0
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		value += int(buf[0]&0x7F) * multiplier
		if buf[0]&0x80 == 0 {
			break
		}
		multiplier *= 128
		if multiplier > 128*128*128 {
			return nil, fmt.Errorf("malformed remaining length")
		}
	}
	header.RemainingLen = value

	return header, nil
}

// ReadString reads a UTF-8 encoded string (length-prefixed)
func ReadString(r io.Reader) (string, error) {
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return "", err
	}
	length := binary.BigEndian.Uint16(lenBuf)

	if length == 0 {
		return "", nil
	}

	strBuf := make([]byte, length)
	if _, err := io.ReadFull(r, strBuf); err != nil {
		return "", err
	}
	return string(strBuf), nil
}

// WriteString writes a UTF-8 encoded string (length-prefixed)
func WriteString(s string) []byte {
	buf := make([]byte, 2+len(s))
	binary.BigEndian.PutUint16(buf, uint16(len(s)))
	copy(buf[2:], s)
	return buf
}

// DecodeConnectPacket decodes a CONNECT packet
func DecodeConnectPacket(r io.Reader, remainingLen int) (*ConnectPacket, error) {
	pkt := &ConnectPacket{}

	// Read protocol name
	protocolName, err := ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read protocol name: %w", err)
	}
	pkt.ProtocolName = protocolName

	// Read protocol version
	versionBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, versionBuf); err != nil {
		return nil, err
	}
	pkt.ProtocolVersion = versionBuf[0]

	// Read connect flags
	flagsBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, flagsBuf); err != nil {
		return nil, err
	}
	flags := flagsBuf[0]
	pkt.UsernameFlag = (flags & 0x80) > 0
	pkt.PasswordFlag = (flags & 0x40) > 0
	pkt.WillRetain = (flags & 0x20) > 0
	pkt.WillQoS = (flags >> 3) & 0x03
	pkt.WillFlag = (flags & 0x04) > 0
	pkt.CleanSession = (flags & 0x02) > 0

	// Read keep alive
	keepAliveBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, keepAliveBuf); err != nil {
		return nil, err
	}
	pkt.KeepAlive = binary.BigEndian.Uint16(keepAliveBuf)

	// Read client ID
	clientID, err := ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read client ID: %w", err)
	}
	pkt.ClientID = clientID

	// Read will topic and message if present
	if pkt.WillFlag {
		pkt.WillTopic, err = ReadString(r)
		if err != nil {
			return nil, err
		}
		willMsgLen := make([]byte, 2)
		if _, err := io.ReadFull(r, willMsgLen); err != nil {
			return nil, err
		}
		msgLen := binary.BigEndian.Uint16(willMsgLen)
		pkt.WillMessage = make([]byte, msgLen)
		if _, err := io.ReadFull(r, pkt.WillMessage); err != nil {
			return nil, err
		}
	}

	// Read username if present
	if pkt.UsernameFlag {
		pkt.Username, err = ReadString(r)
		if err != nil {
			return nil, err
		}
	}

	// Read password if present
	if pkt.PasswordFlag {
		pwdLen := make([]byte, 2)
		if _, err := io.ReadFull(r, pwdLen); err != nil {
			return nil, err
		}
		passLen := binary.BigEndian.Uint16(pwdLen)
		pkt.Password = make([]byte, passLen)
		if _, err := io.ReadFull(r, pkt.Password); err != nil {
			return nil, err
		}
	}

	return pkt, nil
}

// DecodePublishPacket decodes a PUBLISH packet
func DecodePublishPacket(r io.Reader, header *FixedHeader) (*PublishPacket, error) {
	pkt := &PublishPacket{
		Dup:    (header.Flags & 0x08) > 0,
		QoS:    (header.Flags >> 1) & 0x03,
		Retain: (header.Flags & 0x01) > 0,
	}

	// Read topic name
	topic, err := ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read topic: %w", err)
	}
	pkt.Topic = topic

	// Read packet ID if QoS > 0
	bytesRead := 2 + len(topic)
	if pkt.QoS > 0 {
		packetIDBuf := make([]byte, 2)
		if _, err := io.ReadFull(r, packetIDBuf); err != nil {
			return nil, err
		}
		pkt.PacketID = binary.BigEndian.Uint16(packetIDBuf)
		bytesRead += 2
	}

	// Read payload (remaining bytes)
	payloadLen := header.RemainingLen - bytesRead
	if payloadLen > 0 {
		pkt.Payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(r, pkt.Payload); err != nil {
			return nil, err
		}
	}

	return pkt, nil
}

// DecodeSubscribePacket decodes a SUBSCRIBE packet
func DecodeSubscribePacket(r io.Reader, remainingLen int) (*SubscribePacket, error) {
	pkt := &SubscribePacket{}

	// Read packet ID
	packetIDBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, packetIDBuf); err != nil {
		return nil, err
	}
	pkt.PacketID = binary.BigEndian.Uint16(packetIDBuf)

	bytesRead := 2

	// Read topic filters
	for bytesRead < remainingLen {
		topic, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read topic filter: %w", err)
		}
		bytesRead += 2 + len(topic)

		// Read QoS
		qosBuf := make([]byte, 1)
		if _, err := io.ReadFull(r, qosBuf); err != nil {
			return nil, err
		}
		bytesRead++

		pkt.Topics = append(pkt.Topics, Subscription{
			Topic: topic,
			QoS:   qosBuf[0],
		})
	}

	return pkt, nil
}

// DecodeUnsubscribePacket decodes an UNSUBSCRIBE packet
func DecodeUnsubscribePacket(r io.Reader, remainingLen int) (*UnsubscribePacket, error) {
	pkt := &UnsubscribePacket{}

	// Read packet ID
	packetIDBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, packetIDBuf); err != nil {
		return nil, err
	}
	pkt.PacketID = binary.BigEndian.Uint16(packetIDBuf)

	bytesRead := 2

	// Read topic filters
	for bytesRead < remainingLen {
		topic, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read topic filter: %w", err)
		}
		bytesRead += 2 + len(topic)

		pkt.Topics = append(pkt.Topics, topic)
	}

	return pkt, nil
}
