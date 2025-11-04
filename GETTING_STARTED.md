# Go Development Quick Start Guide

## ✅ Setup Complete!

Your MQTT-Server Go project is now ready for development.

## 📁 Project Structure

```
MQTT-Server/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── server/
│   │   ├── server.go            # Server implementation
│   │   └── server_test.go       # Server tests
│   ├── mqtt/                    # MQTT protocol handling (TODO)
│   ├── store/
│   │   └── interface.go         # Storage interface
│   └── auth/                    # Authentication (TODO)
├── config/
│   └── config.yaml              # Configuration file
├── go.mod                       # Go module definition
├── Makefile                     # Build automation
├── Dockerfile                   # Container build
└── README.md                    # Project documentation
```

## 🚀 Common Commands

### Run the server:
```powershell
go run ./cmd/server
```

### Build the binary:
```powershell
go build -o mqtt-server.exe ./cmd/server
```

### Run tests:
```powershell
go test ./...
```

### Format code:
```powershell
go fmt ./...
```

### Add dependencies:
```powershell
# Example: Add a YAML parser
go get gopkg.in/yaml.v3

# Update all dependencies
go mod tidy
```

### View module graph:
```powershell
go mod graph
```

## 🔧 VS Code Tips

### Recommended settings.json:
Create `.vscode/settings.json`:
```json
{
    "go.useLanguageServer": true,
    "go.formatTool": "goimports",
    "go.lintTool": "golangci-lint",
    "editor.formatOnSave": true,
    "go.testOnSave": false,
    "go.coverOnSave": false
}
```

### Useful keyboard shortcuts:
- `F5` - Start debugging
- `Ctrl+Shift+B` - Build task
- `Ctrl+Shift+T` - Run tests
- `Shift+Alt+F` - Format document
- `F12` - Go to definition
- `Shift+F12` - Find all references

## 📦 Installing Dependencies

### Popular MQTT libraries:
```powershell
# Mochi-co MQTT (recommended)
go get github.com/mochi-co/mqtt/v2

# Paho MQTT
go get github.com/eclipse/paho.mqtt.golang
```

### Useful packages:
```powershell
# Configuration
go get gopkg.in/yaml.v3

# Logging
go get github.com/sirupsen/logrus
go get go.uber.org/zap

# Database (bbolt)
go get go.etcd.io/bbolt

# Testing
go get github.com/stretchr/testify
```

## 🏃 Next Steps

1. **Test your setup:**
   ```powershell
   go run ./cmd/server
   ```
   You should see: "MQTT Server started successfully on port 1883"
   Press `Ctrl+C` to stop.

2. **Explore Go basics:**
   - Goroutines and channels (concurrency)
   - Interfaces and struct embedding
   - Error handling patterns
   - Testing with `testing` package

3. **Start developing:**
   - Implement MQTT packet parsing in `internal/mqtt/`
   - Add storage implementation in `internal/store/`
   - Implement TLS support in `internal/server/`

4. **Learn Go idioms:**
   - [Effective Go](https://go.dev/doc/effective_go)
   - [Go by Example](https://gobyexample.com/)
   - [Go Tour](https://go.dev/tour/)

## 🐛 Debugging

### Run with debugger in VS Code:
1. Press `F5` or click "Run and Debug"
2. Set breakpoints by clicking left of line numbers
3. Use debug console to inspect variables

### Debug from terminal:
```powershell
# Install delve debugger
go install github.com/go-delve/delve/cmd/dlv@latest

# Run with debugger
dlv debug ./cmd/server
```

## 📚 Essential Go Concepts

### Packages:
- `package main` - Creates an executable
- `package <name>` - Creates a library
- Import paths match directory structure

### Project Layout:
- `cmd/` - Application entry points
- `internal/` - Private packages (cannot be imported by other projects)
- `pkg/` - Public packages (can be imported)
- `config/` - Configuration files

### Go Modules:
- `go.mod` - Module definition and dependencies
- `go.sum` - Checksums for dependency verification
- Always run `go mod tidy` after adding/removing dependencies

## 🎯 Quick Reference

### Create new files:
```powershell
# In VS Code: Right-click folder → New File
# Or use terminal:
New-Item -ItemType File -Path internal\mqtt\packet.go
```

### Run specific test:
```powershell
go test -v -run TestNewServer ./internal/server
```

### Build for different platforms:
```powershell
# Linux
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o mqtt-server ./cmd/server

# macOS
$env:GOOS="darwin"; $env:GOARCH="amd64"; go build -o mqtt-server ./cmd/server

# Windows (default)
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o mqtt-server.exe ./cmd/server
```

## ✅ Verification Checklist

- [x] Go installed and working (`go version`)
- [x] Module initialized (`go.mod` created)
- [x] Project structure created
- [x] Code compiles (`go build ./...`)
- [x] Tests pass (`go test ./...`)
- [x] VS Code Go extension installed

**You're all set! Happy coding! 🎉**
