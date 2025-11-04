package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete server configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	TLS     TLSConfig     `yaml:"tls"`
	Auth    AuthConfig    `yaml:"auth"`
	Storage StorageConfig `yaml:"storage"`
	Limits  LimitsConfig  `yaml:"limits"`
	QoS     QoSConfig     `yaml:"qos"`
	Logging LoggingConfig `yaml:"logging"`
	Metrics MetricsConfig `yaml:"metrics"`
}

// ServerConfig contains server binding and network settings
type ServerConfig struct {
	Host                string        `yaml:"host"`                  // Network interface to bind to
	Port                int           `yaml:"port"`                  // MQTT port (1883 standard)
	KeepAlive           time.Duration `yaml:"keep_alive"`            // Client keep-alive timeout
	WriteTimeout        time.Duration `yaml:"write_timeout"`         // Write operation timeout
	ReadTimeout         time.Duration `yaml:"read_timeout"`          // Read operation timeout
	CleanSessionDefault bool          `yaml:"clean_session_default"` // Default clean session behavior
}

// TLSConfig contains TLS/SSL settings
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`   // Enable TLS
	CertFile string `yaml:"cert_file"` // Server certificate path
	KeyFile  string `yaml:"key_file"`  // Server private key path
	CAFile   string `yaml:"ca_file"`   // CA certificate for client verification
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	Enabled              bool   `yaml:"enabled"`                // Enable authentication
	AllowAnonymous       bool   `yaml:"allow_anonymous"`        // Allow connections without auth
	RequireClientCerts   bool   `yaml:"require_client_certs"`   // Require client certificates (mTLS)
	UsernamePasswordFile string `yaml:"username_password_file"` // Path to username/password file
}

// StorageConfig contains persistence settings
type StorageConfig struct {
	Backend string `yaml:"backend"` // Storage backend: "memory", "bbolt", "redis"
	Path    string `yaml:"path"`    // File path for file-based backends

	// Redis-specific settings (for future use)
	RedisAddr     string `yaml:"redis_addr,omitempty"`
	RedisPassword string `yaml:"redis_password,omitempty"`
	RedisDB       int    `yaml:"redis_db,omitempty"`
}

// LimitsConfig contains connection and message limits
type LimitsConfig struct {
	MaxClients          int   `yaml:"max_clients"`           // Maximum concurrent connections
	MaxMessageSize      int64 `yaml:"max_message_size"`      // Maximum message payload size in bytes
	MaxInflightMessages int   `yaml:"max_inflight_messages"` // Maximum QoS 1/2 messages in flight per client
	RetainedMessages    bool  `yaml:"retained_messages"`     // Enable retained message support
}

// QoSConfig contains Quality of Service settings
type QoSConfig struct {
	MaxQoS        byte          `yaml:"max_qos"`        // Maximum QoS level supported (0, 1, or 2)
	RetryInterval time.Duration `yaml:"retry_interval"` // Retry interval for unacknowledged messages
	MaxRetries    int           `yaml:"max_retries"`    // Maximum retry attempts
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`  // Log level: debug, info, warn, error
	Format string `yaml:"format"` // Log format: text, json
	Output string `yaml:"output"` // Output: stdout, stderr, or file path
}

// MetricsConfig contains Prometheus metrics settings
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"` // Enable metrics endpoint
	Port    int    `yaml:"port"`    // Metrics HTTP server port
	Path    string `yaml:"path"`    // Metrics endpoint path
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults for any missing values
	cfg.setDefaults()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for missing configuration options
func (c *Config) setDefaults() {
	// Server defaults
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 1883
	}
	if c.Server.KeepAlive == 0 {
		c.Server.KeepAlive = 60 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 10 * time.Second
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}

	// Storage defaults
	if c.Storage.Backend == "" {
		c.Storage.Backend = "bbolt"
	}
	if c.Storage.Path == "" {
		c.Storage.Path = "./data/mqtt.db"
	}

	// Limits defaults
	if c.Limits.MaxClients == 0 {
		c.Limits.MaxClients = 1000
	}
	if c.Limits.MaxMessageSize == 0 {
		c.Limits.MaxMessageSize = 256 * 1024 // 256 KB
	}
	if c.Limits.MaxInflightMessages == 0 {
		c.Limits.MaxInflightMessages = 100
	}

	// QoS defaults
	if c.QoS.MaxQoS == 0 {
		c.QoS.MaxQoS = 1
	}
	if c.QoS.RetryInterval == 0 {
		c.QoS.RetryInterval = 10 * time.Second
	}
	if c.QoS.MaxRetries == 0 {
		c.QoS.MaxRetries = 3
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}
	if c.Logging.Output == "" {
		c.Logging.Output = "stdout"
	}

	// Metrics defaults
	if c.Metrics.Port == 0 {
		c.Metrics.Port = 9090
	}
	if c.Metrics.Path == "" {
		c.Metrics.Path = "/metrics"
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate server settings
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Server.Port)
	}

	// Validate TLS settings
	if c.TLS.Enabled {
		if c.TLS.CertFile == "" || c.TLS.KeyFile == "" {
			return fmt.Errorf("TLS enabled but cert_file or key_file not specified")
		}
	}

	// Validate storage backend
	validBackends := map[string]bool{"memory": true, "bbolt": true, "redis": true}
	if !validBackends[c.Storage.Backend] {
		return fmt.Errorf("invalid storage backend: %s (must be memory, bbolt, or redis)", c.Storage.Backend)
	}

	// Validate QoS level
	if c.QoS.MaxQoS > 2 {
		return fmt.Errorf("invalid max_qos: %d (must be 0, 1, or 2)", c.QoS.MaxQoS)
	}

	// Validate log level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	// Validate metrics port
	if c.Metrics.Enabled {
		if c.Metrics.Port < 1 || c.Metrics.Port > 65535 {
			return fmt.Errorf("invalid metrics port: %d (must be 1-65535)", c.Metrics.Port)
		}
		if c.Metrics.Port == c.Server.Port {
			return fmt.Errorf("metrics port cannot be the same as server port")
		}
	}

	return nil
}
