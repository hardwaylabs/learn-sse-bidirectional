# Server-Sent Events & Bidirectional Communication Explained

## Overview

Server-Sent Events (SSE) combined with HTTP POST creates a powerful pattern for **bidirectional real-time communication** between
servers and clients. This is the foundation of how MCP (Model Context Protocol) sampling works.

## What We Built & Tested

### 1. Basic SSE (One-Way Communication)
**Files:** `cmd/simple_sse_server/main.go`, `cmd/simple_sse_client/main.go`

- ✅ **Server → Client**: Events stream from server to client
- ❌ **Client → Server**: No way for client to send data back
- **Use case**: Live updates, notifications, real-time data feeds

### 2. Bidirectional SSE + HTTP (Two-Way Communication)
**Files:** `cmd/bidirectional_server/main.go`, `cmd/bidirectional_client/main.go`

- ✅ **Server → Client**: Requests sent via SSE stream
- ✅ **Client → Server**: Responses sent via HTTP POST
- **Use case**: Remote procedure calls, LLM sampling, interactive workflows

## How Bidirectional Communication Works

### The Pattern
```
1. Client establishes SSE connection to server
2. Server sends REQUEST via SSE stream
3. Client receives request and processes it
4. Client sends RESPONSE via HTTP POST
5. Server correlates response with original request
6. Complete request/response cycle achieved
```

### Technical Flow

#### Step 1: Client Connects
```http
GET /events?client_id=demo_client HTTP/1.1
Accept: text/event-stream
Cache-Control: no-cache
```

#### Step 2: Server Streams Request
```
data: {"id":"req_123","method":"analyze","message":"hello"}

```

#### Step 3: Client Processes & Responds
```http
POST /response HTTP/1.1
Content-Type: application/json
Client-ID: demo_client

{"id":"req_123","result":"Analysis complete: hello"}
```

## Why This Pattern Works

### Advantages
1. **Real-time**: SSE provides instant server→client communication
1. **Reliable**: HTTP POST ensures client→server delivery
1. **Stateful**: Server maintains client sessions and correlates requests
1. **Web-friendly**: Works through firewalls and proxies
1. **Scalable**: Can handle multiple concurrent client connections

### vs. Alternatives
- **WebSockets**: More complex, requires upgrade handshake, can be blocked by proxies
- **Polling**: Inefficient, high latency, resource intensive
- **Long Polling**: Complex connection management, not real-time

## MCP Sampling Connection

This **exact pattern** is what MCP uses for sampling:

### MCP Sampling Flow
1. MCP Client connects with sampling capability (SSE)
1. Someone calls a tool that needs LLM help
1. MCP Server sends sampling request via SSE
1. MCP Client receives request and calls LLM API
1. MCP Client sends LLM response via HTTP POST
1. MCP Server returns result to original tool caller

### Why mcp-go is Broken
- ✅ **Our example**: Clean SSE delivery, events arrive instantly
- ❌ **mcp-go**: SSE connection errors, "context deadline exceeded"
- **Root cause**: mcp-go's StreamableHTTP SSE implementation has bugs

## Key Technical Concepts

### Server-Sent Events (SSE)
- **Protocol**: Text-based streaming over HTTP
- **Format**: `data: message\n\n`
- **Headers**: `Content-Type: text/event-stream`, `Cache-Control: no-cache`
- **Connection**: Long-lived HTTP connection
- **Auto-reconnect**: Browsers automatically reconnect on disconnection

### HTTP POST for Responses
- **Why not SSE both ways?**: SSE is server→client only
- **Reliability**: POST ensures delivery confirmation
- **Correlation**: Request ID links responses to original requests
- **Session management**: Client ID identifies which client responded

### Session & State Management
- **Client Registration**: Server tracks active clients
- **Request Correlation**: IDs match requests with responses
- **Timeout Handling**: Servers can timeout waiting for responses
- **Connection Lifecycle**: Proper cleanup when clients disconnect

## Real-World Applications

### 1. LLM Integration (Like MCP)
- Server needs LLM analysis
- Streams request to client with LLM access
- Client calls OpenAI/Anthropic/etc.
- Returns results via HTTP

### 2. Remote Code Execution
- IDE sends code execution request
- Remote server executes and streams results
- Client can send additional input during execution

### 3. Interactive Workflows
- Server orchestrates multi-step processes
- Clients provide human input or external data
- Each step communicated via SSE→POST pattern

### 4. Distributed Computing
- Coordinator distributes tasks via SSE
- Workers POST results back
- Real-time job status and load balancing

## Implementation Best Practices

### Server Side
- **Connection Management**: Track client connections and clean up properly
- **Request Correlation**: Use unique IDs to match requests/responses
- **Timeout Handling**: Don't wait forever for client responses
- **Error Recovery**: Handle client disconnections gracefully
- **Buffering**: Queue messages if client temporarily disconnects

### Client Side
- **Reconnection Logic**: Handle SSE disconnections and reconnect
- **Request Processing**: Parse SSE messages properly
- **Error Handling**: Send error responses for failed requests
- **State Management**: Track pending requests and responses

### Protocol Design
- **Message Format**: Use JSON for structured data
- **Error Codes**: Standardize error reporting
- **Versioning**: Plan for protocol evolution
- **Authentication**: Secure both SSE and HTTP endpoints

## Code Examples Walkthrough

### Basic SSE Server Key Points
```go
// Essential SSE headers
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")

// SSE message format
fmt.Fprintf(w, "data: %s\n\n", message)

// Critical: Flush immediately
if flusher, ok := w.(http.Flusher); ok {
    flusher.Flush()
}
```

### Bidirectional Server Key Points
```go
// Track clients and their request channels
clients   map[string]chan Request
responses map[string]chan Response

// Send request, wait for response with timeout
respChan := server.sendRequest(clientID, req)
select {
case response := <-respChan:
    // Got response!
case <-time.After(30 * time.Second):
    // Timeout
}
```

### Client Key Points
```go
// Parse SSE stream
scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    if strings.HasPrefix(line, "data: ") {
        data := strings.TrimPrefix(line, "data: ")
        // Process the data
    }
}

// Send response via POST
req, _ := http.NewRequest("POST", "/response", jsonData)
req.Header.Set("Client-ID", clientID)
```

## Testing & Debugging

### What Working SSE Looks Like
- Clean connection establishment (HTTP 200)
- Steady event delivery with no timeouts
- Proper connection lifecycle management
- Immediate event processing

### Common Issues
- **Missing Flush**: Events don't appear until connection closes
- **Wrong Headers**: Browser/client rejects SSE stream
- **Connection Timeouts**: Network/proxy issues
- **Memory Leaks**: Not cleaning up disconnected clients
- **Race Conditions**: Request/response correlation bugs

### Debug Techniques
1. **Browser DevTools**: Network tab shows SSE connections
1. **curl Testing**: Test SSE endpoint directly
1. **Connection Logging**: Log connect/disconnect events
1. **Message Tracing**: Log all sent/received messages
1. **Load Testing**: Verify behavior under concurrent connections

## Further Reading

### SSE Fundamentals
- **[MDN: Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)** - Comprehensive SSE reference
- **[HTML5 SSE Specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)** - Official protocol spec
- **[SSE vs WebSockets](https://stackoverflow.com/questions/5195452/websockets-vs-server-sent-events-eventsource)** - When to use each

### Real-Time Communication Patterns
- **[The Twelve-Factor App: Port Binding](https://12factor.net/port-binding)** - Service architecture principles
- **[Building Scalable Real-Time Systems](https://engineering.linkedin.com/kafka/running-kafka-scale)** - Large-scale real-time patterns
- **[Event-Driven Architecture](https://martinfowler.com/articles/201701-event-driven.html)** - Martin Fowler's guide

### HTTP & Networking
- **[HTTP/2 and Server Push](https://developers.google.com/web/fundamentals/performance/http2)** - Modern HTTP features
- **[Long Polling vs SSE vs WebSockets](https://codeburst.io/polling-vs-sse-vs-websocket-how-to-choose-the-right-one-1859e4e13bd9)** - Comparison guide
- **[Designing Data-Intensive Applications](https://dataintensive.net/)** - Chapter 11 on stream processing

### Go-Specific Resources
- **[Go SSE Implementation Guide](https://www.sohamkamani.com/golang/server-sent-events/)** - Practical Go examples
- **[Building Real-time Apps in Go](https://tutorialedge.net/golang/go-websocket-tutorial/)** - Go real-time patterns
- **[Effective Go: Concurrency](https://golang.org/doc/effective_go#concurrency)** - Go concurrency patterns

### MCP & Protocol Design
- **[Model Context Protocol Spec](https://modelcontextprotocol.io/)** - Official MCP documentation
- **[JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)** - Protocol used by MCP
- **[API Design Patterns](https://cloud.google.com/apis/design)** - Google's API design guide

## What You've Learned

By building and testing these examples, you now understand:

1. ✅ **How SSE works** - Protocol, headers, connection management
1. ✅ **Bidirectional patterns** - SSE + HTTP POST for two-way communication
1. ✅ **Real-time architecture** - When and how to use these patterns
1. ✅ **MCP sampling mechanics** - Exactly how LLM sampling should work
1. ✅ **Debugging techniques** - How to identify and fix connection issues
1. ✅ **Production considerations** - Scaling, reliability, error handling

You're now equipped to:
- **Debug MCP issues** with deep protocol knowledge
- **Contribute to mcp-go** with working reference implementations
- **Design real-time systems** using proven patterns
- **Build LLM integrations** that actually work

## Standards vs. Arbitrary Design Decisions

### What's Defined by Standards

#### ✅ **SSE Specification Requirements**
```http
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**Message Format:**
```
data: your message here\n\n
```

**Connection Pattern:**
- Must be HTTP GET request
- Server keeps connection open
- Events pushed as they occur

#### ✅ **HTTP Standards**
- POST method for client→server responses
- JSON content negotiation
- Standard status codes (200, 404, 500, etc.)

### What's Completely Arbitrary

#### ❌ **Endpoint Names (I Made These Up)**
- `/events` - could be `/stream`, `/notifications`, `/sse`
- `/response` - could be `/callback`, `/reply`, `/answer`  
- `/trigger` - could be `/send`, `/request`, `/invoke`

#### ❌ **URL Structure**
- Query parameters: `?client_id=demo&message=hello`
- Path parameters: `/events/demo_client`
- Headers: `Client-ID: demo_client`

#### ❌ **Message Content Structure**
```json
{"id":"req_123","method":"analyze","message":"hello"}
```
This JSON structure is entirely custom - no standard defines this.

### Real-World Endpoint Examples

**GitHub API:**
```
GET /notifications/events
```

**Discord API:**
```
GET /gateway
```

**Slack RTM API:**
```
GET /rtm.connect
```

**MCP StreamableHTTP:**
```
GET /mcp (with Accept: text/event-stream)
POST /mcp (for responses)
```

### How MCP Actually Works

MCP uses **single endpoint with content negotiation**:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    if r.Header.Get("Accept") == "text/event-stream" {
        // Start SSE stream for server→client
        handleSSEStream(w, r)
    } else if r.Method == "POST" {
        // Handle JSON-RPC for client→server
        handleJSONRPC(w, r)
    }
}
```

### What Actually Defines SSE Communication

#### 1. **Client Detection Pattern**
```http
GET /your-endpoint HTTP/1.1
Accept: text/event-stream
Cache-Control: no-cache
```

#### 2. **Server Response Pattern**
```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache

data: {"message": "hello"}\n\n
```

#### 3. **Making It Bidirectional**
SSE is **unidirectional** (server→client). Bidirectional requires combining:
- **SSE stream**: Server→Client requests
- **HTTP POST**: Client→Server responses  
- **Correlation IDs**: Link requests with responses

### Why I Used Separate Endpoints

#### Educational Benefits:
- **Clear separation** - easy to understand each part
- **Debugging clarity** - different URLs in logs
- **Conceptual isolation** - see SSE vs HTTP POST distinctly

#### Production Reality:
- **MCP uses single endpoint** - more elegant
- **Content negotiation** - based on headers
- **Simpler routing** - fewer endpoints to manage

### Standards References

#### What IS Standardized:
- **[HTML5 SSE Specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)** - Message format, headers
- **[HTTP/1.1 RFC](https://tools.ietf.org/html/rfc7231)** - POST methods, status codes
- **[JSON-RPC 2.0](https://www.jsonrpc.org/specification)** - Used by MCP for message structure

#### What is NOT Standardized:
- Endpoint URL patterns
- Authentication methods
- Message content structure (beyond SSE format)
- Error handling specifics
- Reconnection behavior
- Session management

### Key Insight

The **protocol mechanics** matter, not the URL paths. Whether you use:
- `/events` + `/response` (my teaching example)
- `/mcp` with content negotiation (MCP's approach)  
- `/api/stream` + `/api/callback` (your choice)

The core SSE + HTTP POST pattern works the same way! 🎯

## Conclusion

The SSE + HTTP POST pattern is a powerful, proven approach for bidirectional real-time communication. Your working examples
demonstrate that the pattern itself is solid - the issue with MCP sampling is purely in the mcp-go library's implementation.

With this foundation, you can confidently contribute to the mcp-go project and build robust real-time applications! 🚀