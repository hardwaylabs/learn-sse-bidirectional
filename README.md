# Learn SSE & Bidirectional Communication

A hands-on learning repository for Server-Sent Events (SSE) and bidirectional real-time communication patterns.

## What You'll Learn

1. **SSE Fundamentals** - Protocol mechanics, headers, connection lifecycle
2. **Bidirectional Patterns** - Combining SSE + HTTP POST for two-way communication  
3. **Real-time Architecture** - When and how to use these patterns
4. **Production Considerations** - Scaling, reliability, error handling

## Learning Path

### 1. Basic SSE - One-Way Communication
Start with `go/cmd/basic_sse_server` and `go/cmd/basic_sse_client` to understand:
- SSE protocol fundamentals
- Server→client event streaming
- Connection management
- Browser compatibility

### 2. Bidirectional Communication
Progress to `go/cmd/bidirectional_server` and `go/cmd/bidirectional_client` to explore:
- Two-way request/response patterns
- SSE + HTTP POST combination
- Session correlation
- Real-time workflows

### 3. Real-World Applications
Explore practical use cases in `use-cases/`:
- Chat systems and messaging
- Live dashboards and monitoring
- Remote execution and control

## Quick Start

### Prerequisites
- Go 1.21 or later
- Basic understanding of HTTP

### Run Basic SSE Example

```bash
# Terminal 1: Start the server
cd go
go run cmd/basic_sse_server/main.go

# Terminal 2: Connect with client
go run cmd/basic_sse_client/main.go

# Terminal 3: Test with browser
open http://localhost:8080/web
```

### Run Bidirectional Example

```bash
# Terminal 1: Start bidirectional server
go run cmd/bidirectional_server/main.go

# Terminal 2: Connect bidirectional client
go run cmd/bidirectional_client/main.go

# Terminal 3: Trigger requests
go run cmd/demo_trigger/main.go
```

## Key Concepts

### Server-Sent Events (SSE)
- **Protocol**: Text-based streaming over HTTP
- **Headers**: `Content-Type: text/event-stream`, `Cache-Control: no-cache`
- **Format**: `data: message\n\n`
- **Connection**: Long-lived HTTP connection with auto-reconnect

### Bidirectional Communication Pattern
```
1. Client establishes SSE connection to server
2. Server sends REQUEST via SSE stream
3. Client receives request and processes it
4. Client sends RESPONSE via HTTP POST
5. Server correlates response with original request
6. Complete request/response cycle achieved
```

### Why This Pattern Works
- **Real-time**: SSE provides instant server→client communication
- **Reliable**: HTTP POST ensures client→server delivery
- **Web-friendly**: Works through firewalls and proxies
- **Scalable**: Handles multiple concurrent connections

## Directory Structure

```
learn-sse-bidirectional/
├── README.md                    # This file
├── CONCEPTS.md                  # Detailed theory and explanations
├── go/                          # Go implementation
│   ├── cmd/
│   │   ├── basic_sse_server/    # Simple SSE server
│   │   ├── basic_sse_client/    # Simple SSE client
│   │   ├── bidirectional_server/# Full bidirectional server
│   │   ├── bidirectional_client/# Full bidirectional client
│   │   └── demo_trigger/        # Request trigger tool
│   ├── web/
│   │   └── sse-test.html        # Browser SSE test page
│   ├── go.mod                   # Go module definition
│   └── README.md                # Go-specific instructions
├── python/                      # Python implementation (coming soon)
│   └── README.md                # Python placeholder
└── use-cases/                   # Real-world examples
    ├── CHAT_SYSTEM.md          # Chat system architecture
    ├── LIVE_DASHBOARD.md       # Dashboard patterns
    └── REMOTE_EXECUTION.md     # Remote execution patterns
```

## Implementations

### Go Implementation (Complete)
The Go implementation provides fully working examples with:
- Clean SSE connection handling
- Proper session management
- Request/response correlation
- Error handling and timeouts
- Browser testing interface

### Python Implementation (Planned)
Coming soon - the same patterns implemented in Python using:
- FastAPI for HTTP/SSE
- asyncio for concurrency
- WebSocket alternatives comparison

## Real-World Applications

### 1. LLM Integration Systems
- Server requests LLM analysis via SSE
- Client processes using OpenAI/Anthropic APIs
- Results returned via HTTP POST
- **Example**: Model Context Protocol (MCP) sampling

### 2. Remote Code Execution
- IDE sends execution requests
- Remote server runs code and streams output
- Client provides input during execution
- **Example**: Cloud development environments

### 3. Interactive Workflows
- Server orchestrates multi-step processes
- Clients provide human input or external data
- Real-time progress updates
- **Example**: CI/CD pipelines with approval steps

### 4. Live Monitoring & Dashboards
- Server streams metrics and events
- Clients display real-time visualizations
- Bidirectional control actions
- **Example**: Infrastructure monitoring systems

## Best Practices

### Server Implementation
- Track client connections and clean up properly
- Use unique IDs to correlate requests/responses
- Implement reasonable timeouts
- Handle client disconnections gracefully
- Queue messages for temporarily disconnected clients

### Client Implementation
- Handle SSE disconnections and reconnect automatically
- Parse SSE messages correctly
- Send error responses for failed requests
- Maintain request state properly

### Production Considerations
- Use HTTPS for security
- Implement authentication on both endpoints
- Add rate limiting and circuit breakers
- Monitor connection health and metrics
- Plan for horizontal scaling

## Debugging Tips

### Working SSE Connection Signs
- Clean HTTP 200 connection establishment
- Steady event delivery without timeouts
- Proper connection lifecycle management
- Events appear immediately in client

### Common Issues
- **Missing Flush**: Events delayed until connection closes
- **Wrong Headers**: Browser/client rejects SSE stream  
- **Network Timeouts**: Proxy or firewall interference
- **Memory Leaks**: Not cleaning up disconnected clients
- **Race Conditions**: Request/response correlation bugs

### Debug Tools
1. **Browser DevTools**: Network tab shows SSE connections
2. **curl**: Test SSE endpoints directly
3. **Connection Logs**: Monitor connect/disconnect events
4. **Message Tracing**: Log all sent/received messages

## Related Projects

- [learn-mcp-sampling](https://github.com/pavelanni/learn-mcp-sampling) - See these concepts applied to MCP protocol
- [Caret Labs](https://caretlabs.dev) - Hands-on learning methodology for developers

## Contributing

This is a learning resource! Feel free to:
- Add examples in other programming languages
- Improve documentation and explanations
- Share real-world use cases
- Report issues or suggest improvements

## License

MIT License - Use these examples freely for learning and building!

---

**Happy Learning!** 🚀

*Part of the Caret Labs learning methodology - hands-on, progressive, multi-component education for developers.*