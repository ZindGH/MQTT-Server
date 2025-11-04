# MQTT-Server

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A lightweight, production-ready MQTT broker implementation in Go with TLS client certificate authentication and pluggable persistence layer.

## 🎯 Overview

This MQTT broker is designed for small to medium-scale deployments (hundreds of clients) with enterprise-grade security and extensibility in mind. Built with Go's excellent concurrency model, it provides a solid foundation for IoT applications requiring reliable message delivery.

### Key Features

- **🔐 Secure by Default**: TLS client certificate authentication (mutual TLS)
- **💾 Pluggable Persistence**: Interface-based storage abstraction (file-based, Redis, PostgreSQL, RocksDB)
- **📊 Observable**: Prometheus metrics integration
- **🎛️ QoS Support**: QoS 0, 1, and planned QoS 2 (exactly-once delivery)
- **🔄 Persistent Sessions**: Client state and offline message queueing
- **📡 Standard Compliant**: Full MQTT 3.1.1 protocol support
- **🏗️ Modular Architecture**: Clean interfaces for easy component replacement

### Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.21+ |
| Authentication | TLS Client Certificates (mTLS) |
| Persistence | Pluggable (default: bbolt embedded DB) |
| Metrics | Prometheus |
| Deployment | Docker, Native Binary |

## 🚀 Quick Start

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.21+** - [Download here](https://go.dev/dl/)
- **OpenSSL** - For TLS certificate generation
- **Git** - For version control
- **Docker** (optional) - For containerized deployment

### Installation

#### Option 1: Native Build

```bash
# Clone the repository
git clone https://github.com/ZindGH/MQTT-Server.git
cd MQTT-Server

# Build the project
go build ./...

# Run the server (uses config/config.yaml by default)
go run ./cmd/server
```

#### Option 2: Docker

```bash
# Build Docker image
docker build -t mqtt-server:latest .

# Run container (exposes port 1883)
docker run --rm -p 1883:1883 mqtt-server:latest
```

### Configuration

The broker reads its configuration from `config/config.yaml`. Key configuration options include:

- **Bind Address & Port**: Network interface and port settings
- **TLS Certificate Paths**: Server and CA certificate locations
- **Persistence Backend**: Storage implementation selection
- **Authentication Options**: Client certificate validation settings

Example `config/config.yaml`:
```yaml
server:
  host: "0.0.0.0"
  port: 1883
tls:
  enabled: true
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"
  ca_file: "certs/ca.crt"
persistence:
  backend: "bbolt"
  path: "data/mqtt.db"
```

## 🔒 TLS Certificate Setup

The server requires mutual TLS (mTLS) for client authentication. Follow these steps to generate certificates for development:

### Step 1: Create Certificate Authority (CA)

```bash
# Generate CA private key (4096-bit RSA)
openssl genrsa -out ca.key 4096

# Generate self-signed CA certificate (valid for 10 years)
openssl req -new -x509 -days 3650 -key ca.key -subj "/CN=mqtt-dev-ca" -out ca.crt
```

### Step 2: Generate Server Certificate

```bash
# Generate server private key
openssl genrsa -out server.key 2048

# Create certificate signing request
openssl req -new -key server.key -subj "/CN=localhost" -out server.csr

# Sign server certificate with CA
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365
```

### Step 3: Generate Client Certificate

```bash
# Generate client private key
openssl genrsa -out client.key 2048

# Create certificate signing request
openssl req -new -key client.key -subj "/CN=test-client" -out client.csr

# Sign client certificate with CA
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365
```

### Step 4: Configure Server

Update your `config/config.yaml` to reference the generated certificates:
- `server.crt` and `server.key` for server identity
- `ca.crt` for client certificate verification

> ⚠️ **Production Warning**: For production environments, use certificates from a trusted Certificate Authority, implement certificate rotation, and secure private keys with proper access controls.

## 📋 Features

### Protocol Support

- ✅ **MQTT 3.1.1 Packet Types**
  - CONNECT, CONNACK, PUBLISH, PUBACK, PUBREC, PUBREL, PUBCOMP
  - SUBSCRIBE, SUBACK, UNSUBSCRIBE, UNSUBACK
  - PINGREQ, PINGRESP, DISCONNECT

- ✅ **QoS Levels**
  - **QoS 0** (At most once): Fire and forget
  - **QoS 1** (At least once): Acknowledged delivery
  - **QoS 2** (Exactly once): 🚧 Planned after QoS 1 stabilization

### Authentication & Authorization

- ✅ TLS client certificate verification (mTLS)
- 🚧 Pluggable authentication layer (JWT, username/password)
- 🚧 Access Control Lists (ACLs) for topic permissions

### Topic Routing

- ✅ Standard MQTT topic filters
- ✅ Single-level wildcards (`+`)
- ✅ Multi-level wildcards (`#`)

### Persistence & Durability

- ✅ Pluggable storage interface
- ✅ File-based embedded database (bbolt)
- ✅ Retained messages
- ✅ Persistent sessions with offline message queueing
- 🚧 Redis backend implementation
- 🚧 PostgreSQL backend implementation

### Management & Observability

- ✅ Prometheus metrics endpoints
- 🚧 Admin REST API
- 🚧 gRPC management interface

### Testing & CI/CD

- ✅ Unit tests for core components
- ✅ Integration tests with MQTT clients
- ✅ GitHub Actions CI pipeline

> Legend: ✅ Implemented | 🚧 Planned/In Progress

## 📚 MQTT Concepts

### Quality of Service (QoS) Levels

Understanding QoS is crucial for reliable message delivery:

| QoS Level | Name | Description | Use Case |
|-----------|------|-------------|----------|
| **QoS 0** | At most once | Fire and forget, no acknowledgment | Sensor data where occasional loss is acceptable |
| **QoS 1** | At least once | Acknowledged delivery, possible duplicates | Important messages where duplicates can be handled |
| **QoS 2** | Exactly once | 4-way handshake, guaranteed single delivery | Critical commands requiring exactly-once semantics |

#### QoS 1 Flow

```
Publisher → PUBLISH → Broker → PUBLISH → Subscriber
           ← PUBACK ←        ← PUBACK ←
```

The broker persists the message until PUBACK is received. If the connection fails, the message will be redelivered (potential duplicates).

#### QoS 2 Flow (Planned)

```
Publisher → PUBLISH → Broker → PUBLISH → Subscriber
           ← PUBREC ←         ← PUBREC ←
           → PUBREL →         → PUBREL →
           ← PUBCOMP ←        ← PUBCOMP ←
```

Four-way handshake ensures exactly-once delivery by tracking additional state.

### Durable Delivery

**Durable delivery** means messages are persisted to disk/database, surviving broker restarts. This ensures:
- Messages aren't lost during broker failures
- Offline clients receive queued messages on reconnection
- QoS 1/2 guarantees are maintained across restarts

### Persistent Sessions

Persistent sessions enable:
- **Subscription retention**: Client subscriptions survive disconnections
- **Offline message queueing**: Messages accumulate while client is offline
- **Seamless reconnection**: Queued messages delivered when client reconnects with same client ID

This requires persistent storage for session state and message queues.

## 🏗️ Architecture

### Storage Abstraction

The broker uses a pluggable `Store` interface for all persistence operations:

```go
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
}
```

**Implementations:**
- **bbolt** (default): Single-file embedded database, perfect for small deployments
- **Redis** (planned): High-performance in-memory store with persistence
- **PostgreSQL** (planned): Relational database for complex querying
- **RocksDB** (planned): High-performance embedded key-value store

### Project Structure

```
MQTT-Server/
├── cmd/
│   └── server/           # Main application entry point
├── internal/
│   ├── server/          # Server core (networking, TLS, lifecycle)
│   ├── mqtt/            # MQTT protocol handling
│   ├── store/           # Storage interface + implementations
│   │   ├── interface.go
│   │   ├── bbolt/
│   │   └── redis/
│   └── auth/            # Authentication and authorization
├── config/              # Configuration files (YAML)
├── pkg/                 # Public APIs and utilities
├── tools/               # Helper scripts (cert generation, testing)
├── .github/
│   └── workflows/       # CI/CD pipelines
├── Dockerfile
├── go.mod
└── README.md
```

### Clustering Considerations

Clustering involves multiple broker nodes sharing subscription state and message routing. This requires:
- **Shared state**: Distributed database or message bus for synchronization
- **Load balancing**: Client connection distribution
- **Message routing**: Cross-node subscription matching
- **Split-brain prevention**: Consensus algorithms (Raft, etc.)

For small-scale deployments, start with a single node. Clustering can be added later when horizontal scalability is required.

## 🛠️ Development

### Getting Started with Go

If you're new to Go:

```bash
# Initialize module dependencies
go mod download

# Format code automatically
go fmt ./...

# Run tests
go test ./...

# Run with race detector
go run -race ./cmd/server

# Build optimized binary
go build -ldflags="-s -w" -o mqtt-server ./cmd/server
```

### Best Practices

- **Keep packages focused**: Small, single-purpose packages are easier to maintain
- **Use interfaces**: Enable component swapping and testing with mocks
- **Leverage goroutines**: Handle each client connection concurrently
- **Channel communication**: Use channels for safe inter-goroutine communication
- **Error handling**: Always check and handle errors explicitly
- **Testing**: Write unit tests for business logic, integration tests for workflows

### Testing with MQTT Clients

#### Using Mosquitto

```bash
# Subscribe to a topic
mosquitto_sub -h localhost -p 1883 -t "test/topic" \
  --cafile ca.crt --cert client.crt --key client.key

# Publish a message
mosquitto_pub -h localhost -p 1883 -t "test/topic" -m "Hello MQTT" \
  --cafile ca.crt --cert client.crt --key client.key
```

#### Using Paho Python Client

```python
import paho.mqtt.client as mqtt
import ssl

client = mqtt.Client()
client.tls_set(ca_certs="ca.crt", certfile="client.crt", keyfile="client.key")
client.connect("localhost", 1883)
client.publish("test/topic", "Hello from Paho")
```

## 🗺️ Roadmap

### Phase 1: Core Functionality (Current)
- [x] Basic MQTT protocol support (QoS 0, 1)
- [x] TLS with client certificate authentication
- [x] File-based persistence with bbolt
- [x] Retained messages
- [ ] Complete persistent session implementation
- [ ] Prometheus metrics integration

### Phase 2: Enhanced Features
- [ ] QoS 2 (exactly-once delivery)
- [ ] Redis storage backend
- [ ] PostgreSQL storage backend
- [ ] WebSocket transport support
- [ ] Admin REST API
- [ ] JWT authentication support

### Phase 3: Enterprise Features
- [ ] Horizontal clustering
- [ ] Message bridge functionality
- [ ] Advanced ACL system
- [ ] Rate limiting and quotas
- [ ] Audit logging
- [ ] MQTT 5.0 protocol support

## 🤝 Contributing

Contributions are welcome! Here's how you can help:

1. **Fork the repository**
2. **Create a feature branch** (`git checkout -b feature/amazing-feature`)
3. **Commit your changes** (`git commit -m 'Add amazing feature'`)
4. **Push to the branch** (`git push origin feature/amazing-feature`)
5. **Open a Pull Request**

Please ensure:
- Code is formatted with `go fmt`
- All tests pass (`go test ./...`)
- New features include tests
- Documentation is updated

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgements

- [MQTT 3.1.1 Specification](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/mqtt-v3.1.1.html)
- [Mochi-co MQTT](https://github.com/mochi-co/mqtt) - Reference implementation
- [bbolt](https://github.com/etcd-io/bbolt) - Embedded database
- The Go community for excellent tooling and libraries

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/ZindGH/MQTT-Server/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ZindGH/MQTT-Server/discussions)

---

**Built with ❤️ using Go**