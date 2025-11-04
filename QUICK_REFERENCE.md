# MQTT Server - Quick Reference Guide

## 🚀 Quick Commands

### Server

```bash
# Build server
go build -o mqtt-server.exe ./cmd/server

# Run server
./mqtt-server.exe

# Run with custom config
./mqtt-server.exe -config custom.yaml

# Run tests
go test -v ./test/integration/
```

### Demo Client

```bash
# Build client
cd tools/client
go build -o mqtt-client.exe

# Run client
./mqtt-client.exe

# Connect to specific broker
./mqtt-client.exe -broker tcp://192.168.1.100:1883 -client my-id
```

## 📝 Client Commands

| Command | Usage | Example |
|---------|-------|---------|
| Subscribe | `sub <topic> [qos]` | `sub sensors/temp 1` |
| Unsubscribe | `unsub <topic>` | `unsub sensors/temp` |
| Publish | `pub <topic> <msg> [qos] [retain]` | `pub test/msg Hello 1` |
| Status | `status` or `s` | `s` |
| Help | `help` or `h` | `h` |
| Exit | `exit` or `quit` or `q` | `q` |

## 🎯 Wildcard Patterns

### Single-Level (+)
Matches **exactly one** level:
```
sensors/+/temperature
  ✅ sensors/room1/temperature
  ✅ sensors/outdoor/temperature
  ❌ sensors/room1/temp/current (too many levels)
  ❌ sensors/temperature (missing level)
```

### Multi-Level (#)
Matches **zero or more** levels (must be last):
```
home/#
  ✅ home/living/temp
  ✅ home/bedroom/lights/status
  ✅ home (matches zero levels after home/)
  ❌ house/living (different first level)
```

### Mixed Wildcards
Combine both in one pattern:
```
home/+/sensors/#
  ✅ home/living/sensors/temp
  ✅ home/bedroom/sensors/motion/front
  ❌ home/sensors/temp (missing middle level)
```

## 📦 Retained Messages

### Publish Retained Message
```bash
> pub device/status online 0 retain
```

### Clear Retained Message
```bash
> pub device/status "" 0 retain
```

### Subscribe (receives retained immediately)
```bash
> sub device/status 0
📨 Message received:
   Topic: device/status
   Retained: true
   Payload: online
```

## 🎚️ QoS Levels

| QoS | Name | Guarantee | Use Case |
|-----|------|-----------|----------|
| 0 | At most once | Best effort | Non-critical data |
| 1 | At least once | Acknowledged | Important messages |
| 2 | Exactly once | Guaranteed | Critical (not yet implemented) |

### Examples
```bash
# QoS 0 - Fire and forget
> pub sensors/temp 22.5 0

# QoS 1 - With acknowledgment
> pub alerts/fire Emergency! 1

# QoS 2 - Exactly once (coming soon)
> pub payments/transaction 100.00 2
```

## 🧪 Testing Scenarios

### Test 1: Basic Pub/Sub
```bash
# Terminal 1
mqtt-client -client subscriber
> sub test/hello 0

# Terminal 2
mqtt-client -client publisher
> pub test/hello "Hello, World!" 0
```

### Test 2: Wildcard Subscriptions
```bash
# Terminal 1
> sub sensors/+/temperature 1

# Terminal 2
> pub sensors/room1/temperature 22.5 1
> pub sensors/outdoor/temperature 18.0 1
# Both received by subscriber
```

### Test 3: Retained Messages
```bash
# Terminal 1 - Publish retained
> pub status/system running 0 retain

# Terminal 2 - Subscribe later
> sub status/system 0
# Immediately receives "running"
```

### Test 4: Mixed Wildcards
```bash
# Terminal 1
> sub home/+/sensors/# 0

# Terminal 2
> pub home/living/sensors/temp 21.5 0      # ✅ Matches
> pub home/bedroom/sensors/humidity 60 0   # ✅ Matches
> pub home/sensors/temp 20 0               # ❌ Doesn't match
```

## 📊 Test Results

All tests passing (11/11):
```
✅ TestMQTTConnect                 - Basic connection
✅ TestMQTTPublishSubscribe        - Pub/sub messaging
✅ TestMQTTMultipleClients         - 5 concurrent clients
✅ TestMQTTQoS1                    - QoS 1 with PUBACK
✅ TestMQTTPingPong                - Keep-alive
✅ TestMQTTReconnect               - Session persistence
✅ TestMQTTWildcardSubscriptions   - # multi-level wildcard
✅ TestMQTTLargeMessage            - 100KB messages
✅ TestMQTTRetainedMessages        - Retained message lifecycle
✅ TestMQTTSingleLevelWildcard     - + single-level wildcard
✅ TestMQTTMixedWildcards          - Combined + and # wildcards
```

## 🔧 Configuration

Key settings in `config/config.yaml`:

```yaml
server:
  host: "127.0.0.1"
  port: 1883

storage:
  backend: "bbolt"
  path: "./data/mqtt.db"

limits:
  max_clients: 1000
  max_message_size: 256 KB
  retained_messages: true

qos:
  max_qos: 1

metrics:
  enabled: true
  port: 9090
```

## 📖 Additional Documentation

- [ADVANCED_FEATURES.md](ADVANCED_FEATURES.md) - Detailed feature documentation
- [tools/client/README.md](tools/client/README.md) - Demo client guide
- [README.md](README.md) - Full project documentation
- [TEST_DOCUMENTATION.md](TEST_DOCUMENTATION.md) - Testing guide

## 🆘 Troubleshooting

### Can't Connect
```bash
# Check if server is running
netstat -an | findstr 1883

# Verify broker address
mqtt-client -broker tcp://127.0.0.1:1883
```

### Messages Not Received
1. Check topic spelling (case-sensitive)
2. Verify subscription succeeded
3. Use wildcard `#` to debug: `sub # 0`

### Retained Message Not Working
1. Ensure `retain` flag is set: `pub topic msg 0 retain`
2. Subscribe with matching topic filter
3. Check server logs for "Stored retained message"

## 🎯 Pro Tips

1. **Monitor All Traffic**: Subscribe to `#` to see all messages
2. **Test Patterns First**: Use wildcards to test topic structure
3. **Use QoS 1 for Important Data**: Ensures delivery acknowledgment
4. **Clear Retained on Shutdown**: Publish empty retained messages before stopping
5. **Use Descriptive Topics**: `home/living/temp` not `h/l/t`

## 📈 Current Capabilities

- ✅ MQTT 3.1.1 Protocol
- ✅ QoS 0 & 1
- ✅ Single & Multi-level Wildcards
- ✅ Retained Messages
- ✅ Persistent Sessions
- ✅ Keep-Alive (Ping/Pong)
- ✅ Multiple Concurrent Clients
- ✅ Large Messages (100KB+)
- ✅ Prometheus Metrics

## 🚀 Coming Soon

- ⏳ QoS 2 (Exactly-once)
- ⏳ Will Messages
- ⏳ Persistent Retained Messages
- ⏳ WebSocket Support
- ⏳ TLS/SSL Encryption

---

**Version**: 1.1.0  
**Status**: Production-Ready ✨  
**Last Updated**: November 4, 2025
