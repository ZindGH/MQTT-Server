# Server Connection Tests

## Overview

Comprehensive test suite for the MQTT server's connection handling functionality. These tests verify that the server can accept connections, read data, and handle multiple concurrent clients.

## Test Cases

### 1. `TestNewServer`
**Purpose**: Verify server initialization  
**Tests**:
- Server instance creation
- Default port configuration (1883)
- Client map initialization

### 2. `TestServerStartStop`
**Purpose**: Verify server lifecycle management  
**Tests**:
- Server starts without errors
- Server listens on the configured port
- Server stops gracefully
- No goroutine leaks

### 3. `TestConnectionHandling`
**Purpose**: Verify basic connection acceptance and data reading  
**Tests**:
- Server accepts TCP connections
- Connection established successfully
- Data can be written to the connection
- Server reads data without crashing

**Flow**:
```
Client → Connect → Server
Client → Send "Hello MQTT Server" → Server reads 17 bytes
```

### 4. `TestMultipleConnections`
**Purpose**: Verify concurrent connection handling  
**Tests**:
- Server handles multiple simultaneous connections
- Each connection operates independently
- Data from all clients is processed
- No race conditions or deadlocks

**Flow**:
```
5 Clients → Connect → Server
Each Client → Send message → Server reads
All connections handled concurrently
```

### 5. `TestConnectionReadLoop`
**Purpose**: Verify sequential message handling  
**Tests**:
- Server's read loop processes multiple messages
- Messages are received in order
- Connection stays alive between messages

**Flow**:
```
Client → Connect → Server
Client → Send "First message" → Server reads
Client → Send "Second message" → Server reads
Client → Send "Third message" → Server reads
```

### 6. `TestConnectionClose`
**Purpose**: Verify graceful connection termination  
**Tests**:
- Server handles client disconnect gracefully
- Server continues accepting new connections after disconnect
- No server crash on connection close
- `handleConnection` goroutine exits properly

**Flow**:
```
Client1 → Connect → Send data → Disconnect
Server → Still running
Client2 → Connect → Success
```

### 7. `TestLargeDataTransfer`
**Purpose**: Verify handling of data larger than buffer size  
**Tests**:
- Server handles data larger than 1024-byte buffer
- Multiple read iterations work correctly
- No data loss or corruption
- Buffer overflow protection

**Flow**:
```
Client → Send 2048 bytes (2x buffer size)
Server → Reads in chunks of 1024 bytes
All data received successfully
```

## Test Results

All tests pass successfully:
```
✅ TestNewServer           - Server creation
✅ TestServerStartStop     - Lifecycle management
✅ TestConnectionHandling  - Basic connection & read
✅ TestMultipleConnections - Concurrent clients (5)
✅ TestConnectionReadLoop  - Sequential messages (3)
✅ TestConnectionClose     - Graceful disconnect
✅ TestLargeDataTransfer   - Large payload (2048 bytes)
```

## What is Verified

### ✅ Connection Management
- Server accepts TCP connections on port 1883
- `handleConnection` is called for each new connection
- Connections run in separate goroutines

### ✅ Data Reading
- Server reads data from clients
- Buffer handling works correctly (1024 bytes)
- Multiple reads work in sequence
- Large data (>1024 bytes) handled properly

### ✅ Concurrent Operations
- Multiple clients can connect simultaneously
- Each connection is independent
- No race conditions detected
- Thread-safe operations

### ✅ Error Handling
- Connection close (EOF) handled gracefully
- Server doesn't crash on client disconnect
- `handleConnection` goroutine exits cleanly
- Server continues running after errors

### ✅ Resource Management
- Connections are properly closed with `defer`
- Server stops cleanly
- All goroutines terminate
- No resource leaks

## Running the Tests

### Run all tests:
```powershell
go test -v ./internal/server
```

### Run specific test:
```powershell
go test -v -run TestConnectionHandling ./internal/server
```

### Run with race detector:
```powershell
go test -race -v ./internal/server
```

### Run with coverage:
```powershell
go test -cover ./internal/server
```

## Current Limitations

The current implementation:
- ✅ Accepts connections
- ✅ Reads data into buffer
- ✅ Handles multiple connections
- ❌ Does NOT echo data back (receive only)
- ❌ Does NOT parse MQTT packets yet
- ❌ Does NOT track client state yet

## Next Steps

1. **Add Echo Functionality**: Modify `handleConnection` to write data back
2. **MQTT Protocol**: Parse actual MQTT packets (CONNECT, PUBLISH, etc.)
3. **Client Registration**: Store connected clients in the `clients` map
4. **Session Management**: Implement client ID tracking
5. **Subscription Handling**: Add topic subscription logic

## Example: Testing Manually

You can also test the server manually using telnet:

```powershell
# Start the server
go run ./cmd/server

# In another terminal, connect with telnet
telnet localhost 1883

# Type any text and press Enter
# Server will log: "Received X bytes from [address]"
```

Or using PowerShell:

```powershell
$client = New-Object System.Net.Sockets.TcpClient
$client.Connect("localhost", 1883)
$stream = $client.GetStream()
$data = [System.Text.Encoding]::UTF8.GetBytes("Hello Server")
$stream.Write($data, 0, $data.Length)
$client.Close()
```

## Conclusion

All connection tests pass successfully! The `handleConnection` method works correctly and the server can:
- Accept connections ✅
- Read data from clients ✅
- Handle multiple concurrent connections ✅
- Process sequential messages ✅
- Handle large data transfers ✅
- Recover from connection closes ✅

The foundation is solid and ready for MQTT protocol implementation!
