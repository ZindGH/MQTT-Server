package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	broker   = flag.String("broker", "tcp://127.0.0.1:1883", "MQTT broker address")
	clientID = flag.String("client", "demo-client", "Client ID")
	username = flag.String("user", "", "Username for authentication")
	password = flag.String("pass", "", "Password for authentication")
	qos      = flag.Int("qos", 0, "Quality of Service (0, 1, 2)")
)

func main() {
	flag.Parse()

	fmt.Println("╔════════════════════════════════════════════════╗")
	fmt.Println("║      MQTT Demo Client - Interactive Mode      ║")
	fmt.Println("╚════════════════════════════════════════════════╝")
	fmt.Printf("\nConnecting to broker: %s\n", *broker)
	fmt.Printf("Client ID: %s\n", *clientID)
	fmt.Printf("QoS Level: %d\n\n", *qos)

	// Configure MQTT client
	opts := mqtt.NewClientOptions()
	opts.AddBroker(*broker)
	opts.SetClientID(*clientID)
	opts.SetCleanSession(false)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetWriteTimeout(10 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	if *username != "" {
		opts.SetUsername(*username)
	}
	if *password != "" {
		opts.SetPassword(*password)
	}

	// Set up message handler
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("\n📨 Message received:\n")
		fmt.Printf("   Topic: %s\n", msg.Topic())
		fmt.Printf("   QoS: %d\n", msg.Qos())
		fmt.Printf("   Retained: %t\n", msg.Retained())
		fmt.Printf("   Payload: %s\n", string(msg.Payload()))
		fmt.Print("\n> ")
	})

	// Connection status handlers
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		fmt.Println("✅ Connected to MQTT broker")
		fmt.Print("\n> ")
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		fmt.Printf("\n⚠️  Connection lost: %v\n", err)
		fmt.Println("Attempting to reconnect...")
	})

	// Create and connect client
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		fmt.Println("❌ Connection timeout")
		os.Exit(1)
	}
	if token.Error() != nil {
		fmt.Printf("❌ Failed to connect: %v\n", token.Error())
		os.Exit(1)
	}

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\n👋 Disconnecting...")
		client.Disconnect(250)
		os.Exit(0)
	}()

	// Print help
	printHelp()

	// Interactive loop
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print("> ")
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			fmt.Print("> ")
			continue
		}

		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "help", "h":
			printHelp()

		case "subscribe", "sub":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: subscribe <topic> [qos]")
			} else {
				topic := parts[1]
				qosLevel := byte(*qos)
				if len(parts) >= 3 {
					fmt.Sscanf(parts[2], "%d", &qosLevel)
				}

				token := client.Subscribe(topic, qosLevel, nil)
				if token.WaitTimeout(5 * time.Second) {
					if token.Error() != nil {
						fmt.Printf("❌ Subscribe failed: %v\n", token.Error())
					} else {
						fmt.Printf("✅ Subscribed to '%s' (QoS %d)\n", topic, qosLevel)
					}
				} else {
					fmt.Printf("❌ Subscribe timeout for '%s'\n", topic)
				}
			}

		case "unsubscribe", "unsub":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: unsubscribe <topic>")
			} else {
				topic := parts[1]
				token := client.Unsubscribe(topic)
				if token.WaitTimeout(5 * time.Second) {
					if token.Error() != nil {
						fmt.Printf("❌ Unsubscribe failed: %v\n", token.Error())
					} else {
						fmt.Printf("✅ Unsubscribed from '%s'\n", topic)
					}
				} else {
					fmt.Printf("❌ Unsubscribe timeout for '%s'\n", topic)
				}
			}

		case "publish", "pub":
			if len(parts) < 3 {
				fmt.Println("❌ Usage: publish <topic> <message> [qos] [retain]")
			} else {
				topic := parts[1]
				message := strings.Join(parts[2:], " ")

				// Check for QoS and retain flags at the end
				qosLevel := byte(*qos)
				retain := false

				// Parse optional parameters from message
				msgParts := parts[2:]
				endIdx := len(msgParts)

				// Check if last parameter is "retain" or "r"
				if endIdx > 0 && (strings.ToLower(msgParts[endIdx-1]) == "retain" || strings.ToLower(msgParts[endIdx-1]) == "r") {
					retain = true
					endIdx--
				}

				// Check if parameter before last (or last if no retain) is QoS
				if endIdx > 0 {
					if qVal := msgParts[endIdx-1]; qVal == "0" || qVal == "1" || qVal == "2" {
						fmt.Sscanf(qVal, "%d", &qosLevel)
						endIdx--
					}
				}

				// Reconstruct message without QoS and retain flags
				if endIdx < len(msgParts) {
					message = strings.Join(msgParts[:endIdx], " ")
				}

				token := client.Publish(topic, qosLevel, retain, message)
				if token.WaitTimeout(5 * time.Second) {
					if token.Error() != nil {
						fmt.Printf("❌ Publish failed: %v\n", token.Error())
					} else {
						retainStr := ""
						if retain {
							retainStr = " [RETAINED]"
						}
						fmt.Printf("✅ Published to '%s' (QoS %d)%s\n", topic, qosLevel, retainStr)
					}
				} else {
					fmt.Printf("❌ Publish timeout for '%s'\n", topic)
				}
			}

		case "status", "s":
			if client.IsConnected() {
				fmt.Println("✅ Status: Connected")
			} else {
				fmt.Println("❌ Status: Disconnected")
			}

		case "exit", "quit", "q":
			fmt.Println("👋 Disconnecting...")
			client.Disconnect(250)
			return

		default:
			fmt.Printf("❌ Unknown command: %s (type 'help' for available commands)\n", cmd)
		}

		fmt.Print("> ")
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

func printHelp() {
	fmt.Println("\n📖 Available Commands:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  subscribe <topic> [qos]")
	fmt.Println("  sub <topic> [qos]           - Subscribe to a topic")
	fmt.Println()
	fmt.Println("  unsubscribe <topic>")
	fmt.Println("  unsub <topic>               - Unsubscribe from a topic")
	fmt.Println()
	fmt.Println("  publish <topic> <message> [qos] [retain]")
	fmt.Println("  pub <topic> <message> [qos] [retain]")
	fmt.Println("                              - Publish a message")
	fmt.Println()
	fmt.Println("  status / s                  - Show connection status")
	fmt.Println("  help / h                    - Show this help")
	fmt.Println("  exit / quit / q             - Exit the client")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("\n💡 Examples:")
	fmt.Println("  sub sensors/+/temperature 1")
	fmt.Println("  pub sensors/room1/temp 25.5 1")
	fmt.Println("  pub home/status online 0 retain")
	fmt.Println()
}
