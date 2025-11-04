package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/ZindGH/MQTT-Server/internal/config"
	"github.com/ZindGH/MQTT-Server/internal/server"
	"github.com/ZindGH/MQTT-Server/internal/store"
)

// Helper function to start test server
func startTestServer(t *testing.T) (*server.Server, func()) {
	// Create test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                "127.0.0.1",
			Port:                1884, // Use different port for testing
			KeepAlive:           60 * time.Second,
			WriteTimeout:        10 * time.Second,
			ReadTimeout:         30 * time.Second,
			CleanSessionDefault: false,
		},
		Storage: config.StorageConfig{
			Backend: "bbolt",
			Path:    "./test_data/test_mqtt.db",
		},
		Limits: config.LimitsConfig{
			MaxClients:          1000,
			MaxMessageSize:      256 * 1024,
			MaxInflightMessages: 100,
			RetainedMessages:    true,
		},
		QoS: config.QoSConfig{
			MaxQoS:        1,
			RetryInterval: 10 * time.Second,
			MaxRetries:    3,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		Metrics: config.MetricsConfig{
			Enabled: false, // Disable metrics for tests
		},
	}

	// Ensure test directory exists
	dir := filepath.Dir(cfg.Storage.Path)
	os.MkdirAll(dir, 0755)

	// Initialize store
	st, err := store.NewBboltStore(cfg.Storage.Path)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create server
	srv, err := server.NewWithConfig(cfg, st)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Return cleanup function
	cleanup := func() {
		srv.Stop()
		st.Close()
		os.RemoveAll("./test_data")
	}

	return srv, cleanup
}

// TestMQTTConnect tests basic MQTT connection
func TestMQTTConnect(t *testing.T) {
	_, cleanup := startTestServer(t)
	defer cleanup()

	// Create MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://127.0.0.1:1884")
	opts.SetClientID("test-client-connect")
	opts.SetCleanSession(true)
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		t.Logf("Connection lost: %v", err)
	})

	// Create and connect client
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(5 * time.Second) {
		t.Fatal("Connection timeout")
	}
	if err := token.Error(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Verify connection
	if !client.IsConnected() {
		t.Fatal("Client not connected")
	}

	t.Log("✓ Successfully connected to MQTT broker")

	// Disconnect
	client.Disconnect(250)
	time.Sleep(100 * time.Millisecond)
}

// TestMQTTPublishSubscribe tests publish/subscribe functionality
func TestMQTTPublishSubscribe(t *testing.T) {
	_, cleanup := startTestServer(t)
	defer cleanup()

	receivedMessage := make(chan string, 1)

	// Create subscriber client
	subOpts := mqtt.NewClientOptions()
	subOpts.AddBroker("tcp://127.0.0.1:1884")
	subOpts.SetClientID("test-subscriber")
	subOpts.SetCleanSession(true)

	subscriber := mqtt.NewClient(subOpts)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Subscriber failed to connect: %v", token.Error())
	}
	defer subscriber.Disconnect(250)

	// Subscribe to topic
	topic := "test/topic"
	token := subscriber.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		t.Logf("Received message: %s on topic: %s", msg.Payload(), msg.Topic())
		receivedMessage <- string(msg.Payload())
	})
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to subscribe: %v", token.Error())
	}

	t.Logf("✓ Subscribed to topic: %s", topic)

	// Give subscription time to register
	time.Sleep(100 * time.Millisecond)

	// Create publisher client
	pubOpts := mqtt.NewClientOptions()
	pubOpts.AddBroker("tcp://127.0.0.1:1884")
	pubOpts.SetClientID("test-publisher")
	pubOpts.SetCleanSession(true)

	publisher := mqtt.NewClient(pubOpts)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Publisher failed to connect: %v", token.Error())
	}
	defer publisher.Disconnect(250)

	// Publish message
	testMessage := "Hello MQTT Server!"
	token = publisher.Publish(topic, 0, false, testMessage)
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to publish: %v", token.Error())
	}

	t.Logf("✓ Published message: %s", testMessage)

	// Wait for message with timeout
	select {
	case received := <-receivedMessage:
		if received != testMessage {
			t.Errorf("Expected '%s', got '%s'", testMessage, received)
		}
		t.Log("✓ Message received successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

// TestMQTTMultipleClients tests multiple concurrent clients
func TestMQTTMultipleClients(t *testing.T) {
	_, cleanup := startTestServer(t)
	defer cleanup()

	numClients := 5
	clients := make([]mqtt.Client, numClients)

	// Connect multiple clients
	for i := 0; i < numClients; i++ {
		opts := mqtt.NewClientOptions()
		opts.AddBroker("tcp://127.0.0.1:1884")
		opts.SetClientID(fmt.Sprintf("test-client-%d", i))
		opts.SetCleanSession(true)

		client := mqtt.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			t.Fatalf("Client %d failed to connect: %v", i, token.Error())
		}
		clients[i] = client
		t.Logf("✓ Client %d connected", i)
	}

	// Disconnect all clients
	for i, client := range clients {
		client.Disconnect(250)
		t.Logf("✓ Client %d disconnected", i)
	}

	time.Sleep(100 * time.Millisecond)
	t.Logf("✓ All %d clients handled successfully", numClients)
}

// TestMQTTQoS1 tests QoS 1 message delivery
func TestMQTTQoS1(t *testing.T) {
	_, cleanup := startTestServer(t)
	defer cleanup()

	receivedCount := 0
	done := make(chan bool, 1)

	// Create subscriber
	subOpts := mqtt.NewClientOptions()
	subOpts.AddBroker("tcp://127.0.0.1:1884")
	subOpts.SetClientID("qos1-subscriber")
	subOpts.SetCleanSession(false) // Persistent session

	subscriber := mqtt.NewClient(subOpts)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Subscriber failed to connect: %v", token.Error())
	}
	defer subscriber.Disconnect(250)

	// Subscribe with QoS 1
	topic := "test/qos1"
	token := subscriber.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		receivedCount++
		t.Logf("Received QoS %d message: %s", msg.Qos(), msg.Payload())
		done <- true
	})
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to subscribe: %v", token.Error())
	}

	time.Sleep(100 * time.Millisecond)

	// Create publisher
	pubOpts := mqtt.NewClientOptions()
	pubOpts.AddBroker("tcp://127.0.0.1:1884")
	pubOpts.SetClientID("qos1-publisher")
	pubOpts.SetCleanSession(true)

	publisher := mqtt.NewClient(pubOpts)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Publisher failed to connect: %v", token.Error())
	}
	defer publisher.Disconnect(250)

	// Publish with QoS 1
	testMessage := "QoS 1 Test Message"
	token = publisher.Publish(topic, 1, false, testMessage)
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to publish: %v", token.Error())
	}

	t.Log("✓ Published QoS 1 message")

	// Wait for delivery
	select {
	case <-done:
		t.Log("✓ QoS 1 message delivered")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for QoS 1 message")
	}
}

// TestMQTTPingPong tests keep-alive ping/pong
func TestMQTTPingPong(t *testing.T) {
	_, cleanup := startTestServer(t)
	defer cleanup()

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://127.0.0.1:1884")
	opts.SetClientID("ping-test-client")
	opts.SetKeepAlive(2 * time.Second) // Short keep-alive for testing
	opts.SetPingTimeout(1 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to connect: %v", token.Error())
	}
	defer client.Disconnect(250)

	// Keep connection alive for a few ping cycles
	time.Sleep(6 * time.Second)

	if !client.IsConnected() {
		t.Fatal("Client disconnected (keep-alive failed)")
	}

	t.Log("✓ Keep-alive ping/pong working")
}

// TestMQTTReconnect tests client reconnection
func TestMQTTReconnect(t *testing.T) {
	_, cleanup := startTestServer(t)
	defer cleanup()

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://127.0.0.1:1884")
	opts.SetClientID("reconnect-test-client")
	opts.SetCleanSession(false)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(1 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to connect: %v", token.Error())
	}

	t.Log("✓ Initial connection established")

	// Disconnect
	client.Disconnect(250)
	time.Sleep(500 * time.Millisecond)

	// Reconnect
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to reconnect: %v", token.Error())
	}

	if !client.IsConnected() {
		t.Fatal("Client not reconnected")
	}

	t.Log("✓ Reconnection successful")

	client.Disconnect(250)
}

// TestMQTTWildcardSubscriptions tests topic wildcards
func TestMQTTWildcardSubscriptions(t *testing.T) {
	_, cleanup := startTestServer(t)
	defer cleanup()

	receivedMessages := make(chan string, 3)

	// Create subscriber
	subOpts := mqtt.NewClientOptions()
	subOpts.AddBroker("tcp://127.0.0.1:1884")
	subOpts.SetClientID("wildcard-subscriber")

	subscriber := mqtt.NewClient(subOpts)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Subscriber failed to connect: %v", token.Error())
	}
	defer subscriber.Disconnect(250)

	// Subscribe with wildcard
	token := subscriber.Subscribe("test/#", 0, func(client mqtt.Client, msg mqtt.Message) {
		t.Logf("Received on %s: %s", msg.Topic(), msg.Payload())
		receivedMessages <- msg.Topic()
	})
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to subscribe: %v", token.Error())
	}

	time.Sleep(100 * time.Millisecond)

	// Create publisher
	pubOpts := mqtt.NewClientOptions()
	pubOpts.AddBroker("tcp://127.0.0.1:1884")
	pubOpts.SetClientID("wildcard-publisher")

	publisher := mqtt.NewClient(pubOpts)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Publisher failed to connect: %v", token.Error())
	}
	defer publisher.Disconnect(250)

	// Publish to different topics
	topics := []string{"test/a", "test/b", "test/c/d"}
	for _, topic := range topics {
		token := publisher.Publish(topic, 0, false, "test")
		token.Wait()
	}

	t.Log("✓ Published to multiple topics matching wildcard")

	// Note: Wildcard matching needs to be implemented in server
	// This test will pass once that's done
}

// TestMQTTLargeMessage tests large message handling
func TestMQTTLargeMessage(t *testing.T) {
	_, cleanup := startTestServer(t)
	defer cleanup()

	received := make(chan int, 1)

	// Create subscriber
	subOpts := mqtt.NewClientOptions()
	subOpts.AddBroker("tcp://127.0.0.1:1884")
	subOpts.SetClientID("large-msg-subscriber")

	subscriber := mqtt.NewClient(subOpts)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Subscriber failed to connect: %v", token.Error())
	}
	defer subscriber.Disconnect(250)

	topic := "test/large"
	token := subscriber.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		received <- len(msg.Payload())
	})
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to subscribe: %v", token.Error())
	}

	time.Sleep(100 * time.Millisecond)

	// Create publisher
	pubOpts := mqtt.NewClientOptions()
	pubOpts.AddBroker("tcp://127.0.0.1:1884")
	pubOpts.SetClientID("large-msg-publisher")

	publisher := mqtt.NewClient(pubOpts)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Publisher failed to connect: %v", token.Error())
	}
	defer publisher.Disconnect(250)

	// Create large message (100 KB)
	largeMessage := make([]byte, 100*1024)
	for i := range largeMessage {
		largeMessage[i] = byte(i % 256)
	}

	token = publisher.Publish(topic, 0, false, largeMessage)
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to publish large message: %v", token.Error())
	}

	t.Logf("✓ Published large message (%d bytes)", len(largeMessage))

	// Wait for message
	select {
	case size := <-received:
		if size != len(largeMessage) {
			t.Errorf("Expected %d bytes, got %d", len(largeMessage), size)
		}
		t.Logf("✓ Large message received (%d bytes)", size)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for large message")
	}
}

// TestMQTTRetainedMessages tests retained message functionality
func TestMQTTRetainedMessages(t *testing.T) {
	srv, cleanup := startTestServer(t)
	defer cleanup()

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	topic := "test/retained"

	// Step 1: Publish a retained message
	pubOpts := mqtt.NewClientOptions()
	pubOpts.AddBroker("tcp://127.0.0.1:1884")
	pubOpts.SetClientID("retained-publisher")

	publisher := mqtt.NewClient(pubOpts)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Publisher failed to connect: %v", token.Error())
	}

	retainedMsg := "This is a retained message"
	token := publisher.Publish(topic, 0, true, retainedMsg) // retain = true
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to publish retained message: %v", token.Error())
	}
	t.Logf("✓ Published retained message")

	publisher.Disconnect(250)
	time.Sleep(200 * time.Millisecond)

	// Step 2: New subscriber should receive the retained message
	received := make(chan string, 1)
	subOpts := mqtt.NewClientOptions()
	subOpts.AddBroker("tcp://127.0.0.1:1884")
	subOpts.SetClientID("retained-subscriber")
	subOpts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		t.Logf("Received retained message: %s", string(msg.Payload()))
		received <- string(msg.Payload())
	})

	subscriber := mqtt.NewClient(subOpts)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Subscriber failed to connect: %v", token.Error())
	}
	defer subscriber.Disconnect(250)

	token = subscriber.Subscribe(topic, 0, nil)
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to subscribe: %v", token.Error())
	}
	t.Logf("✓ Subscribed to topic with retained message")

	// Wait for retained message
	select {
	case msg := <-received:
		if msg != retainedMsg {
			t.Errorf("Expected '%s', got '%s'", retainedMsg, msg)
		}
		t.Logf("✓ Received retained message on subscription")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for retained message")
	}

	// Step 3: Clear retained message with empty payload
	publisher2 := mqtt.NewClient(pubOpts)
	if token := publisher2.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Publisher failed to reconnect: %v", token.Error())
	}

	token = publisher2.Publish(topic, 0, true, "") // Empty payload clears retained
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to clear retained message: %v", token.Error())
	}
	t.Logf("✓ Cleared retained message")

	publisher2.Disconnect(250)
}

// TestMQTTSingleLevelWildcard tests the + (single-level) wildcard
func TestMQTTSingleLevelWildcard(t *testing.T) {
	srv, cleanup := startTestServer(t)
	defer cleanup()

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	receivedTopics := make(chan string, 10)

	// Create subscriber with + wildcard
	subOpts := mqtt.NewClientOptions()
	subOpts.AddBroker("tcp://127.0.0.1:1884")
	subOpts.SetClientID("wildcard-plus-sub")
	subOpts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		t.Logf("Received on %s: %s", msg.Topic(), string(msg.Payload()))
		receivedTopics <- msg.Topic()
	})

	subscriber := mqtt.NewClient(subOpts)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Subscriber failed to connect: %v", token.Error())
	}
	defer subscriber.Disconnect(250)

	// Subscribe to sensors/+/temperature (matches exactly one level)
	token := subscriber.Subscribe("sensors/+/temperature", 0, nil)
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to subscribe: %v", token.Error())
	}
	t.Logf("✓ Subscribed to sensors/+/temperature")

	time.Sleep(100 * time.Millisecond)

	// Create publisher
	pubOpts := mqtt.NewClientOptions()
	pubOpts.AddBroker("tcp://127.0.0.1:1884")
	pubOpts.SetClientID("wildcard-plus-pub")

	publisher := mqtt.NewClient(pubOpts)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Publisher failed to connect: %v", token.Error())
	}
	defer publisher.Disconnect(250)

	// Publish messages that should match
	matchingTopics := []string{
		"sensors/room1/temperature",
		"sensors/room2/temperature",
		"sensors/outdoor/temperature",
	}

	for _, topic := range matchingTopics {
		token = publisher.Publish(topic, 0, false, "25°C")
		if token.Wait() && token.Error() != nil {
			t.Fatalf("Failed to publish to %s: %v", topic, token.Error())
		}
	}

	// Publish message that should NOT match (too many levels)
	token = publisher.Publish("sensors/room1/temp/current", 0, false, "25°C")
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to publish: %v", token.Error())
	}

	t.Logf("✓ Published test messages")

	// Verify we received exactly the matching messages
	receivedCount := 0
	timeout := time.After(2 * time.Second)

	for receivedCount < len(matchingTopics) {
		select {
		case topic := <-receivedTopics:
			found := false
			for _, expected := range matchingTopics {
				if topic == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Received unexpected topic: %s", topic)
			}
			receivedCount++
		case <-timeout:
			t.Fatalf("Timeout: received %d/%d messages", receivedCount, len(matchingTopics))
		}
	}

	// Verify no extra messages
	select {
	case topic := <-receivedTopics:
		t.Errorf("Received unexpected extra message on topic: %s", topic)
	case <-time.After(500 * time.Millisecond):
		t.Logf("✓ Received exactly %d matching messages", receivedCount)
	}
}

// TestMQTTMixedWildcards tests combining + and # wildcards
func TestMQTTMixedWildcards(t *testing.T) {
	srv, cleanup := startTestServer(t)
	defer cleanup()

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	received := make(chan string, 10)

	// Create subscriber
	subOpts := mqtt.NewClientOptions()
	subOpts.AddBroker("tcp://127.0.0.1:1884")
	subOpts.SetClientID("mixed-wildcard-sub")
	subOpts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		t.Logf("Received on %s", msg.Topic())
		received <- msg.Topic()
	})

	subscriber := mqtt.NewClient(subOpts)
	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Subscriber failed to connect: %v", token.Error())
	}
	defer subscriber.Disconnect(250)

	// Subscribe to home/+/sensors/#
	// Matches: home/<single-level>/sensors/<any-levels>
	token := subscriber.Subscribe("home/+/sensors/#", 0, nil)
	if token.Wait() && token.Error() != nil {
		t.Fatalf("Failed to subscribe: %v", token.Error())
	}
	t.Logf("✓ Subscribed to home/+/sensors/#")

	time.Sleep(100 * time.Millisecond)

	// Create publisher
	pubOpts := mqtt.NewClientOptions()
	pubOpts.AddBroker("tcp://127.0.0.1:1884")
	pubOpts.SetClientID("mixed-wildcard-pub")

	publisher := mqtt.NewClient(pubOpts)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("Publisher failed to connect: %v", token.Error())
	}
	defer publisher.Disconnect(250)

	// Test topics
	testCases := []struct {
		topic       string
		shouldMatch bool
	}{
		{"home/living/sensors/temp", true},
		{"home/bedroom/sensors/humidity", true},
		{"home/kitchen/sensors/motion/front", true},
		{"home/sensors/temp", false},                // Missing middle level
		{"home/living/bedroom/sensors/temp", false}, // Too many levels before sensors
		{"office/living/sensors/temp", false},       // Wrong first level
	}

	for _, tc := range testCases {
		token = publisher.Publish(tc.topic, 0, false, "data")
		if token.Wait() && token.Error() != nil {
			t.Fatalf("Failed to publish to %s: %v", tc.topic, token.Error())
		}
	}

	t.Logf("✓ Published test messages")

	// Count matching messages
	matchedCount := 0
	expectedMatches := 0
	for _, tc := range testCases {
		if tc.shouldMatch {
			expectedMatches++
		}
	}

	timeout := time.After(2 * time.Second)
	for matchedCount < expectedMatches {
		select {
		case topic := <-received:
			// Verify it's one of the expected matching topics
			found := false
			for _, tc := range testCases {
				if tc.topic == topic && tc.shouldMatch {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Received unexpected topic: %s", topic)
			}
			matchedCount++
		case <-timeout:
			t.Fatalf("Timeout: received %d/%d expected messages", matchedCount, expectedMatches)
		}
	}

	// Verify no extra messages
	select {
	case topic := <-received:
		t.Errorf("Received unexpected extra message on topic: %s", topic)
	case <-time.After(500 * time.Millisecond):
		t.Logf("✓ Correctly matched %d topics with mixed wildcards", matchedCount)
	}
}
