package server

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	srv, err := New()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if srv == nil {
		t.Fatal("Server is nil")
	}

	if srv.port != 1883 {
		t.Errorf("Expected port 1883, got %d", srv.port)
	}

	if srv.clients == nil {
		t.Error("Clients map is nil")
	}
}

func TestServerStartStop(t *testing.T) {
	srv, err := New()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop the server
	if err := srv.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Check if Start() returned without error
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Server Start() returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}

func TestConnectionHandling(t *testing.T) {
	srv, err := New()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	defer srv.Stop()

	// Connect to the server
	conn, err := net.Dial("tcp", "localhost:1883")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Test 1: Send data and verify it's received (echo test)
	testData := []byte("Hello MQTT Server")

	// Send data
	n, err := conn.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to connection: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Give server time to process
	time.Sleep(50 * time.Millisecond)

	// Note: Current implementation doesn't echo back, it just reads
	// So we're testing if the server can read the data without crashing
	t.Log("Successfully sent data to server")
}

func TestMultipleConnections(t *testing.T) {
	srv, err := New()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	defer srv.Stop()

	// Create multiple connections
	numConnections := 5
	connections := make([]net.Conn, numConnections)

	for i := 0; i < numConnections; i++ {
		conn, err := net.Dial("tcp", "localhost:1883")
		if err != nil {
			t.Fatalf("Failed to create connection %d: %v", i, err)
		}
		connections[i] = conn
		defer conn.Close()
	}

	// Send data from each connection
	for i, conn := range connections {
		testData := []byte("Message from client " + string(rune(i+'0')))
		_, err := conn.Write(testData)
		if err != nil {
			t.Errorf("Failed to write from connection %d: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)
	t.Logf("Successfully handled %d concurrent connections", numConnections)
}

func TestConnectionReadLoop(t *testing.T) {
	srv, err := New()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "localhost:1883")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send multiple messages in sequence
	messages := [][]byte{
		[]byte("First message"),
		[]byte("Second message"),
		[]byte("Third message"),
	}

	for i, msg := range messages {
		n, err := conn.Write(msg)
		if err != nil {
			t.Fatalf("Failed to write message %d: %v", i, err)
		}
		if n != len(msg) {
			t.Errorf("Message %d: expected to write %d bytes, wrote %d", i, len(msg), n)
		}
		time.Sleep(20 * time.Millisecond) // Small delay between messages
	}

	t.Log("Successfully sent multiple messages through connection")
}

func TestConnectionClose(t *testing.T) {
	srv, err := New()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	defer srv.Stop()

	// Connect
	conn, err := net.Dial("tcp", "localhost:1883")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Send some data
	testData := []byte("Test data before close")
	conn.Write(testData)
	time.Sleep(50 * time.Millisecond)

	// Close connection
	if err := conn.Close(); err != nil {
		t.Errorf("Failed to close connection: %v", err)
	}

	// Give server time to handle the close
	time.Sleep(50 * time.Millisecond)

	// Try to connect again to ensure server is still running
	conn2, err := net.Dial("tcp", "localhost:1883")
	if err != nil {
		t.Fatalf("Server not accepting new connections after client disconnect: %v", err)
	}
	defer conn2.Close()

	t.Log("Server correctly handled connection close and accepts new connections")
}

func TestLargeDataTransfer(t *testing.T) {
	srv, err := New()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	defer srv.Stop()

	conn, err := net.Dial("tcp", "localhost:1883")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create a large buffer (larger than the 1024 byte buffer in handleConnection)
	largeData := bytes.Repeat([]byte("A"), 2048)

	// Send large data
	totalWritten := 0
	for totalWritten < len(largeData) {
		n, err := conn.Write(largeData[totalWritten:])
		if err != nil {
			t.Fatalf("Failed to write large data: %v", err)
		}
		totalWritten += n
		time.Sleep(10 * time.Millisecond)
	}

	if totalWritten != len(largeData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(largeData), totalWritten)
	}

	time.Sleep(100 * time.Millisecond)
	t.Logf("Successfully transferred %d bytes", totalWritten)
}
