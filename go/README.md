# Go Implementation - SSE & Bidirectional Communication

Go implementation of Server-Sent Events and bidirectional communication patterns.

## Quick Start

### Prerequisites
- Go 1.21 or later
- Basic understanding of HTTP and goroutines

### Run Examples

```bash
# Basic SSE (one-way communication)
go run cmd/basic_sse_server/main.go    # Terminal 1
go run cmd/basic_sse_client/main.go    # Terminal 2

# Bidirectional communication  
go run cmd/bidirectional_server/main.go   # Terminal 1
go run cmd/bidirectional_client/main.go   # Terminal 2
go run cmd/demo_trigger/main.go            # Terminal 3

# Browser testing
open http://localhost:8080/web/sse-test.html
```

## Implementation Details

### Basic SSE Server (`cmd/basic_sse_server/main.go`)
- **Purpose**: Demonstrates one-way server→client communication
- **Key Features**: 
  - Proper SSE headers (`Content-Type: text/event-stream`)
  - Immediate flushing for real-time delivery
  - Clean connection management
- **Learning Focus**: SSE protocol fundamentals

### Basic SSE Client (`cmd/basic_sse_client/main.go`)  
- **Purpose**: Connects to SSE server and processes events
- **Key Features**:
  - HTTP client with SSE stream parsing
  - Event processing loop
  - Connection error handling
- **Learning Focus**: Client-side SSE implementation

### Bidirectional Server (`cmd/bidirectional_server/main.go`)
- **Purpose**: Full two-way communication using SSE + HTTP POST
- **Key Features**:
  - Client session management
  - Request/response correlation with unique IDs  
  - Timeout handling for client responses
  - Multiple concurrent client support
- **Learning Focus**: Complete bidirectional patterns

### Bidirectional Client (`cmd/bidirectional_client/main.go`)
- **Purpose**: Processes server requests and sends responses
- **Key Features**:
  - SSE stream processing for incoming requests
  - HTTP POST for sending responses back
  - Request processing simulation
- **Learning Focus**: Client-side bidirectional handling

### Demo Trigger (`cmd/demo_trigger/main.go`)
- **Purpose**: Initiates requests to test bidirectional flow
- **Key Features**:
  - Simple HTTP client to trigger server actions
  - Demonstrates end-to-end request flow
- **Learning Focus**: How external systems initiate workflows

## Key Go Concepts Demonstrated

### HTTP Server Patterns
```go
// Essential SSE headers
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache") 
w.Header().Set("Connection", "keep-alive")

// Immediate event delivery
fmt.Fprintf(w, "data: %s\n\n", message)
if flusher, ok := w.(http.Flusher); ok {
    flusher.Flush()
}
```

### Concurrent Connection Management
```go
// Track multiple clients
clients := make(map[string]chan Request)
mutex := &sync.RWMutex{}

// Goroutine per connection
go func(clientID string) {
    defer cleanup(clientID)
    handleClient(clientID, requestChan)
}(clientID)
```

### Request/Response Correlation
```go
// Generate unique request IDs
requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

// Wait for response with timeout
select {
case response := <-responseChan:
    // Process response
case <-time.After(30 * time.Second):
    // Handle timeout
}
```

## Architecture Patterns

### Session Management
- **Client Registration**: Server tracks active client connections
- **Cleanup**: Proper resource cleanup when clients disconnect  
- **State Isolation**: Each client has independent request/response channels

### Error Handling
- **Connection Failures**: Graceful handling of client disconnects
- **Timeout Management**: Reasonable timeouts for client responses
- **Recovery**: Automatic reconnection on client side

### Concurrency
- **Goroutine per Client**: Isolated handling for each connection
- **Channel Communication**: Type-safe message passing
- **Mutex Protection**: Thread-safe shared state management

## Browser Testing

### Web Interface (`web/sse-test.html`)
- **Real-time Connection**: Live SSE connection in browser
- **Event Display**: Visual representation of server events
- **Network Tab**: Inspect SSE connections in Developer Tools

### Testing Workflow
1. Start any SSE server (`basic-sse-server` or `bidirectional-server`)
2. Open `http://localhost:8080/web/sse-test.html`
3. Watch events appear in real-time
4. Use browser DevTools to inspect connection

## Common Issues & Solutions

### Events Not Appearing
**Problem**: Events sent but not received by client
**Solution**: Ensure immediate flushing after each event
```go
if flusher, ok := w.(http.Flusher); ok {
    flusher.Flush()
}
```

### Connection Timeouts
**Problem**: SSE connections timing out
**Solution**: Send periodic keepalive events or adjust timeout settings

### Memory Leaks
**Problem**: Server memory grows with disconnected clients  
**Solution**: Implement proper cleanup in goroutine defer statements

### Race Conditions
**Problem**: Concurrent access to client maps
**Solution**: Use mutex protection around shared state

## Production Considerations

### Performance
- **Connection Limits**: Monitor concurrent connection count
- **Memory Usage**: Clean up disconnected clients promptly
- **CPU Usage**: Avoid blocking operations in event handlers

### Security
- **Input Validation**: Sanitize all client inputs
- **Rate Limiting**: Prevent abuse of endpoints
- **Authentication**: Secure both SSE and POST endpoints

### Reliability  
- **Health Checks**: Monitor connection health
- **Circuit Breakers**: Handle downstream failures gracefully
- **Monitoring**: Log connection events and errors

## Next Steps

### Extend These Examples
- Add authentication middleware
- Implement rate limiting
- Add metrics and monitoring
- Support WebSocket fallback

### Real-World Applications
- **Chat Systems**: Multi-user messaging
- **Live Dashboards**: Real-time metrics display
- **Remote Execution**: Code execution with live output
- **Collaborative Tools**: Multi-user document editing

### Integration with Other Projects
- **MCP Implementation**: See [learn-mcp-sampling](https://github.com/pavelanni/learn-mcp-sampling)
- **Python Version**: Compare with Python implementations
- **Production Deployment**: Scale to production environments

## Dependencies

This implementation uses only Go standard library:
- `net/http` - HTTP server and client
- `encoding/json` - JSON message handling
- `sync` - Concurrency primitives
- `time` - Timeouts and timestamps
- `bufio` - Stream processing

No external dependencies needed for core functionality!

---

**Happy Coding!** 🚀

*Clean, idiomatic Go patterns for real-time communication.*