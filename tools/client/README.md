# MQTT Demo Client

An interactive command-line MQTT client for testing and demonstrating the MQTT-Server broker.

## Features

- 📡 **Interactive Mode**: Real-time command-line interface
- 🔄 **Auto-reconnect**: Automatically reconnects on connection loss
- 🎯 **Wildcard Support**: Subscribe to topics with `+` and `#` wildcards
- 📦 **Retained Messages**: Publish messages with the retain flag
- 🎚️ **QoS Levels**: Support for QoS 0, 1, and 2
- 🔐 **Authentication**: Username/password authentication support

## Installation

```bash
# From the MQTT-Server root directory
cd tools/client
go build -o mqtt-client.exe
```

Or build and run directly:
```bash
go run main.go
```

## Usage

### Basic Usage

```bash
# Connect to local broker
mqtt-client

# Connect to specific broker
mqtt-client -broker tcp://localhost:1883

# Connect with client ID
mqtt-client -broker tcp://localhost:1883 -client my-client-id

# Connect with authentication
mqtt-client -broker tcp://broker.example.com:1883 -user myuser -pass mypass

# Set default QoS level
mqtt-client -qos 1
```

### Command-Line Options

| Flag       | Description                  | Default              |
|------------|------------------------------|----------------------|
| `-broker`  | MQTT broker address          | `tcp://127.0.0.1:1883` |
| `-client`  | Client ID                    | `demo-client`        |
| `-user`    | Username for authentication  | (none)               |
| `-pass`    | Password for authentication  | (none)               |
| `-qos`     | Default Quality of Service   | `0`                  |

## Interactive Commands

Once connected, you can use the following commands:

### Subscribe to Topics

```
subscribe <topic> [qos]
sub <topic> [qos]
```

**Examples:**
```
sub sensors/temperature
sub sensors/+/temperature 1
sub home/# 2
```

### Unsubscribe from Topics

```
unsubscribe <topic>
unsub <topic>
```

**Examples:**
```
unsub sensors/temperature
unsub home/#
```

### Publish Messages

```
publish <topic> <message> [qos] [retain]
pub <topic> <message> [qos] [retain]
```

**Examples:**
```
pub sensors/temperature 25.5
pub sensors/room1/temp 22.3 1
pub home/status online 0 retain
pub alerts/fire Emergency! 2 r
```

### Other Commands

| Command          | Description                |
|------------------|----------------------------|
| `status` or `s`  | Show connection status     |
| `help` or `h`    | Display help information   |
| `exit` or `quit` or `q` | Exit the client    |

## Usage Examples

### Example 1: Basic Pub/Sub

**Terminal 1 - Subscriber:**
```
$ mqtt-client -client subscriber1
> sub test/messages 0
✅ Subscribed to 'test/messages' (QoS 0)
```

**Terminal 2 - Publisher:**
```
$ mqtt-client -client publisher1
> pub test/messages "Hello, MQTT!"
✅ Published to 'test/messages' (QoS 0)
```

**Terminal 1 Output:**
```
📨 Message received:
   Topic: test/messages
   QoS: 0
   Retained: false
   Payload: Hello, MQTT!
```

### Example 2: Wildcard Subscriptions

Subscribe to all temperature sensors:
```
> sub sensors/+/temperature 1
✅ Subscribed to 'sensors/+/temperature' (QoS 1)
```

This will match:
- `sensors/room1/temperature`
- `sensors/outdoor/temperature`
- `sensors/kitchen/temperature`

Subscribe to all home topics:
```
> sub home/# 0
✅ Subscribed to 'home/#' (QoS 0)
```

This will match:
- `home/living/temp`
- `home/bedroom/lights/status`
- `home/kitchen/appliances/oven`

### Example 3: Retained Messages

Publish a retained status message:
```
> pub device/status online 0 retain
✅ Published to 'device/status' (QoS 0) [RETAINED]
```

New subscribers to `device/status` will immediately receive "online" when they connect.

Clear a retained message:
```
> pub device/status "" 0 retain
✅ Published to 'device/status' (QoS 0) [RETAINED]
```

### Example 4: QoS Testing

**QoS 0 (At most once):**
```
> pub test/qos0 "Fire and forget" 0
```

**QoS 1 (At least once):**
```
> pub test/qos1 "Acknowledged delivery" 1
```

**QoS 2 (Exactly once):**
```
> pub test/qos2 "Guaranteed delivery" 2
```

### Example 5: Multiple Subscriptions

```
> sub sensors/temperature 0
✅ Subscribed to 'sensors/temperature' (QoS 0)

> sub sensors/humidity 1
✅ Subscribed to 'sensors/humidity' (QoS 1)

> sub home/+/lights 0
✅ Subscribed to 'home/+/lights' (QoS 0)
```

## Wildcard Patterns

### Single-Level Wildcard (+)

Matches **exactly one** topic level:

| Pattern              | Matches                  | Doesn't Match            |
|----------------------|--------------------------|--------------------------|
| `sensors/+/temp`     | `sensors/room1/temp`     | `sensors/room1/a/temp`   |
|                      | `sensors/outdoor/temp`   | `sensors/temp`           |

### Multi-Level Wildcard (#)

Matches **zero or more** topic levels (must be last):

| Pattern              | Matches                  | Doesn't Match            |
|----------------------|--------------------------|--------------------------|
| `home/#`             | `home/living/temp`       | `house/living/temp`      |
|                      | `home/bedroom/lights`    |                          |
|                      | `home/kitchen/a/b/c`     |                          |

### Mixed Wildcards

You can combine both wildcards:

```
home/+/sensors/#
```

Matches:
- `home/living/sensors/temp`
- `home/bedroom/sensors/humidity`
- `home/kitchen/sensors/motion/front`

Doesn't match:
- `home/sensors/temp` (missing middle level)
- `office/living/sensors/temp` (wrong first level)

## Testing Your MQTT Server

### 1. Start the MQTT Server

```bash
# In the MQTT-Server root directory
go run ./cmd/server
```

### 2. Open Multiple Terminals

**Terminal 1 - Monitor all messages:**
```bash
mqtt-client -client monitor
> sub # 0
```

**Terminal 2 - Temperature sensor:**
```bash
mqtt-client -client temp-sensor
> pub sensors/room1/temperature 22.5 1
> pub sensors/room1/temperature 23.0 1
```

**Terminal 3 - Subscribe to sensors:**
```bash
mqtt-client -client sensor-sub
> sub sensors/+/temperature 1
```

### 3. Test Retained Messages

**Publisher:**
```bash
mqtt-client -client publisher
> pub system/status running 0 retain
```

**New Subscriber (will receive retained message immediately):**
```bash
mqtt-client -client new-sub
> sub system/status 0
📨 Message received:
   Topic: system/status
   Retained: true
   Payload: running
```

## Troubleshooting

### Connection Failed

```
❌ Failed to connect: Network Error
```

**Solutions:**
- Verify the broker is running: `netstat -an | findstr 1883`
- Check the broker address: `-broker tcp://127.0.0.1:1883`
- Check firewall settings

### Messages Not Received

**Check:**
1. Verify subscription was successful
2. Check topic spelling (case-sensitive)
3. Verify QoS levels match expectations
4. Use wildcard `#` to monitor all messages

### Connection Lost

The client will automatically attempt to reconnect:
```
⚠️  Connection lost: connection refused
Attempting to reconnect...
✅ Connected to MQTT broker
```

## Building for Distribution

### Windows
```bash
go build -o mqtt-client.exe
```

### Linux
```bash
GOOS=linux GOARCH=amd64 go build -o mqtt-client
```

### macOS
```bash
GOOS=darwin GOARCH=amd64 go build -o mqtt-client
```

## License

Part of the MQTT-Server project. See main project LICENSE for details.
