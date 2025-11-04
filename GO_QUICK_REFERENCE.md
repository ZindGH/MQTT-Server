# 🚀 Go Quick Reference for Your MQTT Project

## Essential Commands

| Command | Description |
|---------|-------------|
| `go run ./cmd/server` | Run the server without building |
| `go build ./cmd/server` | Build executable |
| `go test ./...` | Run all tests |
| `go fmt ./...` | Format all code |
| `go mod tidy` | Clean up dependencies |
| `go get <package>` | Add a dependency |

## VS Code Shortcuts

| Shortcut | Action |
|----------|--------|
| `F5` | Start debugging |
| `Ctrl+Shift+B` | Build |
| `Ctrl+Shift+T` | Run tests |
| `Shift+Alt+F` | Format code |
| `F12` | Go to definition |
| `Ctrl+.` | Quick fix |

## Common Go Patterns

### Error Handling
```go
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### Goroutines (Concurrent Execution)
```go
go func() {
    // This runs concurrently
    doSomething()
}()
```

### Channels (Communication Between Goroutines)
```go
ch := make(chan string)
go func() {
    ch <- "message"  // Send
}()
value := <-ch  // Receive
```

### Defer (Cleanup)
```go
file, err := os.Open("file.txt")
if err != nil {
    return err
}
defer file.Close()  // Runs when function exits
```

### Struct Methods
```go
type Server struct {
    port int
}

func (s *Server) Start() error {
    // s is the receiver
    return nil
}
```

### Interfaces
```go
type Store interface {
    Save(key, value string) error
    Load(key string) (string, error)
}
```

## Project-Specific Tips

### Adding New Packages
1. Create directory in `internal/` or `pkg/`
2. Create `.go` file with `package <name>`
3. Import: `import "github.com/ZindGH/MQTT-Server/internal/<name>"`

### Testing
```go
func TestSomething(t *testing.T) {
    result := MyFunction()
    if result != expected {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

### Configuration Loading
```go
import "gopkg.in/yaml.v3"

type Config struct {
    Port int `yaml:"port"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    err = yaml.Unmarshal(data, &cfg)
    return &cfg, err
}
```

## Useful Go Tools

### Install helpful tools:
```powershell
# Code formatting
go install golang.org/x/tools/cmd/goimports@latest

# Linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Debugger
go install github.com/go-delve/delve/cmd/dlv@latest
```

## Resources

- **Official Docs**: https://go.dev/doc/
- **Go by Example**: https://gobyexample.com/
- **Effective Go**: https://go.dev/doc/effective_go
- **Go Tour**: https://go.dev/tour/

## Next Steps for Your MQTT Server

1. ✅ **Setup complete!** - Project structure ready
2. 📦 **Add dependencies** - Install MQTT library or bbolt
3. 🔧 **Implement MQTT protocol** - Parse packets in `internal/mqtt/`
4. 💾 **Add persistence** - Implement Store interface with bbolt
5. 🔒 **Add TLS** - Configure mTLS in `internal/server/`
6. ✅ **Write tests** - Add tests as you develop
7. 📊 **Add metrics** - Prometheus integration
8. 📝 **Documentation** - Keep README updated

Happy coding! 🎉
