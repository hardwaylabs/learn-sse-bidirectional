# Use Case: Real-Time Chat System

How to build a scalable chat system using SSE + HTTP POST bidirectional patterns.

## Architecture Overview

```
┌─────────────┐    SSE     ┌──────────────┐    HTTP POST    ┌─────────────┐
│   Client A  │◄──────────►│ Chat Server  │◄───────────────►│   Client B  │
└─────────────┘            └──────────────┘                 └─────────────┘
                                   │
                            ┌─────────────┐
                            │  Message    │
                            │  Storage    │
                            └─────────────┘
```

## Why SSE + HTTP POST for Chat?

### Advantages
- **Real-time Delivery**: Messages appear instantly via SSE
- **Reliable Sending**: HTTP POST ensures message delivery confirmation
- **Web-Friendly**: Works through firewalls, no WebSocket upgrade issues
- **Simple Fallback**: Easy to add polling fallback for problematic networks
- **Stateful Server**: Server maintains room membership and message history

### vs. WebSockets
- **Simpler Protocol**: No upgrade handshake complexity
- **Better Debugging**: Standard HTTP requests in network tools
- **Proxy Friendly**: Many corporate proxies block WebSocket upgrades
- **Easier Authentication**: Standard HTTP auth on both directions

### vs. Polling
- **Efficient**: No constant polling overhead
- **Real-time**: Instant message delivery
- **Scalable**: Server pushes only when needed

## Implementation Pattern

### Client Side
```javascript
// 1. Establish SSE connection for receiving messages
const eventSource = new EventSource(`/chat/room/${roomId}/events?user_id=${userId}`);
eventSource.onmessage = function(event) {
    const message = JSON.parse(event.data);
    displayMessage(message);
};

// 2. Send messages via HTTP POST
async function sendMessage(text) {
    await fetch('/chat/messages', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'User-ID': userId
        },
        body: JSON.stringify({
            room_id: roomId,
            message: text,
            timestamp: Date.now()
        })
    });
}
```

### Server Side (Go Example)
```go
// SSE endpoint for receiving messages
func handleChatEvents(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("user_id")
    roomID := mux.Vars(r)["roomId"]
    
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    
    // Add client to room
    clientChan := chatServer.AddClient(roomID, userID)
    defer chatServer.RemoveClient(roomID, userID)
    
    // Stream messages to client
    for message := range clientChan {
        fmt.Fprintf(w, "data: %s\n\n", message.JSON())
        w.(http.Flusher).Flush()
    }
}

// HTTP POST endpoint for sending messages
func handleSendMessage(w http.ResponseWriter, r *http.Request) {
    var msg ChatMessage
    json.NewDecoder(r.Body).Decode(&msg)
    
    // Broadcast to all clients in room
    chatServer.BroadcastToRoom(msg.RoomID, msg)
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode({"status": "sent"})
}
```

## Message Flow

### Sending a Message
```
1. User types message in client
2. Client sends HTTP POST to /chat/messages
3. Server receives and validates message
4. Server broadcasts to all room members via SSE
5. All clients receive message instantly
6. Original sender gets delivery confirmation
```

### Real-time Features
```go
// Typing indicators
type TypingEvent struct {
    UserID   string `json:"user_id"`
    RoomID   string `json:"room_id"`
    IsTyping bool   `json:"is_typing"`
}

// User presence
type PresenceEvent struct {
    UserID string `json:"user_id"`
    Status string `json:"status"` // online, away, offline
}

// Message reactions  
type ReactionEvent struct {
    MessageID string `json:"message_id"`
    UserID    string `json:"user_id"`
    Reaction  string `json:"reaction"` // 👍, ❤️, 😂, etc.
}
```

## Scaling Considerations

### Single Server
```go
type ChatServer struct {
    rooms   map[string]map[string]chan ChatMessage // roomID -> userID -> channel
    mutex   sync.RWMutex
    storage MessageStorage
}

func (cs *ChatServer) BroadcastToRoom(roomID string, msg ChatMessage) {
    cs.mutex.RLock()
    defer cs.mutex.RUnlock()
    
    if room, exists := cs.rooms[roomID]; exists {
        for userID, clientChan := range room {
            select {
            case clientChan <- msg:
                // Message sent successfully
            default:
                // Client buffer full, remove them
                cs.removeSlowClient(roomID, userID)
            }
        }
    }
}
```

### Multi-Server with Redis
```go
// Pub/Sub for multi-server message distribution
type RedisEventBus struct {
    client *redis.Client
}

func (bus *RedisEventBus) PublishMessage(roomID string, msg ChatMessage) {
    bus.client.Publish(fmt.Sprintf("chat:room:%s", roomID), msg.JSON())
}

func (bus *RedisEventBus) SubscribeToRoom(roomID string) <-chan ChatMessage {
    sub := bus.client.Subscribe(fmt.Sprintf("chat:room:%s", roomID))
    msgChan := make(chan ChatMessage)
    
    go func() {
        for msg := range sub.Channel() {
            var chatMsg ChatMessage
            json.Unmarshal([]byte(msg.Payload), &chatMsg)
            msgChan <- chatMsg
        }
    }()
    
    return msgChan
}
```

## Advanced Features

### Message History
```go
func handleRoomHistory(w http.ResponseWriter, r *http.Request) {
    roomID := mux.Vars(r)["roomId"]
    limit := r.URL.Query().Get("limit") // default 50
    before := r.URL.Query().Get("before") // pagination
    
    messages := chatServer.storage.GetRoomHistory(roomID, limit, before)
    json.NewEncoder(w).Encode(messages)
}
```

### File Sharing
```go
func handleFileUpload(w http.ResponseWriter, r *http.Request) {
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()
    
    // Store file and create message
    fileURL := fileStorage.Store(file, header.Filename)
    
    msg := ChatMessage{
        Type:    "file",
        FileURL: fileURL,
        FileName: header.Filename,
        // ... other fields
    }
    
    chatServer.BroadcastToRoom(msg.RoomID, msg)
}
```

### User Authentication
```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        
        user, err := validateJWT(token)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Add user to request context
        ctx := context.WithValue(r.Context(), "user", user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Client Implementation Examples

### React Hook
```javascript
import { useState, useEffect } from 'react';

function useChatRoom(roomId, userId) {
    const [messages, setMessages] = useState([]);
    const [connected, setConnected] = useState(false);
    
    useEffect(() => {
        const eventSource = new EventSource(
            `/chat/room/${roomId}/events?user_id=${userId}`
        );
        
        eventSource.onopen = () => setConnected(true);
        eventSource.onclose = () => setConnected(false);
        eventSource.onmessage = (event) => {
            const message = JSON.parse(event.data);
            setMessages(prev => [...prev, message]);
        };
        
        return () => eventSource.close();
    }, [roomId, userId]);
    
    const sendMessage = async (text) => {
        await fetch('/chat/messages', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                room_id: roomId,
                user_id: userId,
                message: text
            })
        });
    };
    
    return { messages, connected, sendMessage };
}
```

### Vue.js Component
```javascript
// ChatRoom.vue
<template>
  <div class="chat-room">
    <div class="messages" ref="messagesContainer">
      <div v-for="msg in messages" :key="msg.id" class="message">
        <strong>{{ msg.username }}:</strong> {{ msg.text }}
      </div>
    </div>
    <form @submit.prevent="sendMessage">
      <input v-model="newMessage" placeholder="Type message..." />
      <button type="submit">Send</button>
    </form>
  </div>
</template>

<script>
export default {
  data() {
    return {
      messages: [],
      newMessage: '',
      eventSource: null
    };
  },
  mounted() {
    this.connectToChat();
  },
  methods: {
    connectToChat() {
      this.eventSource = new EventSource(`/chat/room/${this.roomId}/events`);
      this.eventSource.onmessage = (event) => {
        this.messages.push(JSON.parse(event.data));
        this.$nextTick(() => this.scrollToBottom());
      };
    },
    async sendMessage() {
      if (!this.newMessage.trim()) return;
      
      await fetch('/chat/messages', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          room_id: this.roomId,
          message: this.newMessage
        })
      });
      
      this.newMessage = '';
    },
    scrollToBottom() {
      const container = this.$refs.messagesContainer;
      container.scrollTop = container.scrollHeight;
    }
  }
};
</script>
```

## Performance Optimization

### Connection Management
```go
type ConnectionManager struct {
    connections map[string]*Connection
    mutex      sync.RWMutex
    cleanup    *time.Ticker
}

func (cm *ConnectionManager) CleanupStaleConnections() {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    
    for userID, conn := range cm.connections {
        if time.Since(conn.lastActivity) > 5*time.Minute {
            conn.close()
            delete(cm.connections, userID)
        }
    }
}
```

### Message Rate Limiting
```go
type RateLimiter struct {
    clients map[string]*rate.Limiter
    mutex   sync.RWMutex
}

func (rl *RateLimiter) Allow(userID string) bool {
    rl.mutex.RLock()
    limiter, exists := rl.clients[userID]
    rl.mutex.RUnlock()
    
    if !exists {
        rl.mutex.Lock()
        limiter = rate.NewLimiter(10, 20) // 10 messages/sec, burst of 20
        rl.clients[userID] = limiter
        rl.mutex.Unlock()
    }
    
    return limiter.Allow()
}
```

## Testing Strategy

### Load Testing
```bash
# Test concurrent connections
for i in {1..100}; do
    curl -H "Accept: text/event-stream" \
         "http://localhost:8080/chat/room/test/events?user_id=user_$i" &
done

# Test message throughput
ab -n 1000 -c 50 -H "Content-Type: application/json" \
   -p message.json http://localhost:8080/chat/messages
```

### Integration Testing
```go
func TestChatFlow(t *testing.T) {
    server := NewChatServer()
    
    // Simulate two clients joining room
    client1 := server.AddClient("room1", "user1")
    client2 := server.AddClient("room1", "user2")
    
    // Send message from client1
    msg := ChatMessage{RoomID: "room1", UserID: "user1", Text: "Hello!"}
    server.BroadcastToRoom("room1", msg)
    
    // Verify client2 receives message
    select {
    case received := <-client2:
        assert.Equal(t, "Hello!", received.Text)
    case <-time.After(time.Second):
        t.Fatal("Message not received")
    }
}
```

## Production Deployment

### Docker Setup
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o chat-server ./cmd/chat-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/chat-server .
CMD ["./chat-server"]
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chat-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: chat-server
  template:
    metadata:
      labels:
        app: chat-server
    spec:
      containers:
      - name: chat-server
        image: chat-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: REDIS_URL
          value: "redis://redis-service:6379"
```

## Security Considerations

### Input Validation
```go
func validateChatMessage(msg *ChatMessage) error {
    if len(msg.Text) > 1000 {
        return errors.New("message too long")
    }
    if strings.TrimSpace(msg.Text) == "" {
        return errors.New("message cannot be empty")
    }
    // Sanitize HTML
    msg.Text = html.EscapeString(msg.Text)
    return nil
}
```

### Rate Limiting & Abuse Prevention
```go
func rateLimitMiddleware(next http.Handler) http.Handler {
    limiter := NewRateLimiter()
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := getUserID(r)
        
        if !limiter.Allow(userID) {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

---

**Real-Time Chat Made Simple** 💬

*Using proven SSE + HTTP POST patterns for reliable, scalable messaging.*