package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Simple request/response system over SSE + HTTP POST (like MCP)
type Request struct {
	ID      string `json:"id"`
	Method  string `json:"method"`
	Message string `json:"message"`
}

type Response struct {
	ID     string `json:"id"`
	Result string `json:"result"`
}

type Server struct {
	clients   map[string]chan Request
	responses map[string]chan Response
	mutex     sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		clients:   make(map[string]chan Request),
		responses: make(map[string]chan Response),
	}
}

func (s *Server) addClient(clientID string) chan Request {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	requestChan := make(chan Request, 10)
	s.clients[clientID] = requestChan
	s.responses[clientID] = make(chan Response, 10)
	
	log.Printf("Client connected: %s", clientID)
	return requestChan
}

func (s *Server) removeClient(clientID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if ch, exists := s.clients[clientID]; exists {
		close(ch)
		delete(s.clients, clientID)
	}
	if ch, exists := s.responses[clientID]; exists {
		close(ch)
		delete(s.responses, clientID)
	}
	
	log.Printf("Client disconnected: %s", clientID)
}

func (s *Server) sendRequest(clientID string, req Request) <-chan Response {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	if clientChan, exists := s.clients[clientID]; exists {
		clientChan <- req
		return s.responses[clientID]
	}
	return nil
}

func main() {
	server := NewServer()

	// SSE endpoint for client to receive requests
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		clientID := r.URL.Query().Get("client_id")
		if clientID == "" {
			clientID = fmt.Sprintf("client_%d", time.Now().Unix())
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		requestChan := server.addClient(clientID)
		defer server.removeClient(clientID)

		// Send client ID first
		fmt.Fprintf(w, "data: {\"type\":\"client_id\",\"id\": \"%s\"}\n\n", clientID)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Stream requests to client
		for {
			select {
			case req, ok := <-requestChan:
				if !ok {
					return
				}
				
				reqJSON, _ := json.Marshal(req)
				fmt.Fprintf(w, "data: %s\n\n", reqJSON)
				
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				
			case <-r.Context().Done():
				return
			}
		}
	})

	// HTTP POST endpoint for client responses
	http.HandleFunc("/response", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var response Response
		if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		clientID := r.Header.Get("Client-ID")
		
		server.mutex.RLock()
		if respChan, exists := server.responses[clientID]; exists {
			respChan <- response
		}
		server.mutex.RUnlock()

		w.WriteHeader(http.StatusOK)
		log.Printf("Received response from %s: %s", clientID, response.Result)
	})

	// Test endpoint to trigger a request
	http.HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		clientID := r.URL.Query().Get("client_id")
		message := r.URL.Query().Get("message")
		
		if clientID == "" || message == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Need client_id and message parameters")
			return
		}

		req := Request{
			ID:      fmt.Sprintf("req_%d", time.Now().Unix()),
			Method:  "analyze",
			Message: message,
		}

		log.Printf("Sending request to client %s: %s", clientID, message)
		
		// Send request and wait for response
		respChan := server.sendRequest(clientID, req)
		if respChan == nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Client not found")
			return
		}

		// Wait for response with timeout
		select {
		case response := <-respChan:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			log.Printf("Got response: %s", response.Result)
			
		case <-time.After(30 * time.Second):
			w.WriteHeader(http.StatusRequestTimeout)
			fmt.Fprint(w, "Request timeout")
			log.Println("Request timed out")
		}
	})

	// Status page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head><title>Bidirectional SSE Server</title></head>
<body>
	<h1>Bidirectional SSE + HTTP Server</h1>
	<p>This demonstrates MCP-style bidirectional communication:</p>
	<ul>
		<li><strong>SSE Stream:</strong> Server → Client requests</li>
		<li><strong>HTTP POST:</strong> Client → Server responses</li>
	</ul>
	<p><strong>Endpoints:</strong></p>
	<ul>
		<li><code>GET /events?client_id=test</code> - SSE stream for client</li>
		<li><code>POST /response</code> - Client sends responses</li>
		<li><code>GET /trigger?client_id=test&message=hello</code> - Trigger request</li>
	</ul>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	})

	log.Println("🚀 Bidirectional SSE Server starting on :8082")
	log.Println("📱 Open browser: http://localhost:8082")
	log.Println("🔗 SSE endpoint: http://localhost:8082/events?client_id=test")
	log.Println("📤 Trigger: http://localhost:8082/trigger?client_id=test&message=hello")
	
	log.Fatal(http.ListenAndServe(":8082", nil))
}