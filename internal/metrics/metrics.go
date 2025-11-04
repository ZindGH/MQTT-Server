package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ClientsConnected tracks the number of currently connected clients
	ClientsConnected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mqtt_clients_connected",
		Help: "Number of currently connected MQTT clients",
	})

	// MessagesReceived counts total messages received
	MessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mqtt_messages_received_total",
			Help: "Total number of MQTT messages received by type",
		},
		[]string{"type"},
	)

	// MessagesSent counts total messages sent
	MessagesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mqtt_messages_sent_total",
			Help: "Total number of MQTT messages sent by type",
		},
		[]string{"type"},
	)

	// BytesReceived tracks bytes received
	BytesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mqtt_bytes_received_total",
		Help: "Total bytes received from MQTT clients",
	})

	// BytesSent tracks bytes sent
	BytesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mqtt_bytes_sent_total",
		Help: "Total bytes sent to MQTT clients",
	})

	// ConnectionsTotal tracks total connection attempts
	ConnectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mqtt_connections_total",
		Help: "Total number of connection attempts",
	})

	// SubscriptionsActive tracks active subscriptions
	SubscriptionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mqtt_subscriptions_active",
		Help: "Number of active subscriptions",
	})

	// RetainedMessages tracks retained messages
	RetainedMessages = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mqtt_retained_messages",
		Help: "Number of retained messages",
	})

	// QoSMessagesInflight tracks in-flight QoS 1/2 messages
	QoSMessagesInflight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mqtt_qos_messages_inflight",
			Help: "Number of in-flight QoS 1/2 messages",
		},
		[]string{"qos"},
	)
)
