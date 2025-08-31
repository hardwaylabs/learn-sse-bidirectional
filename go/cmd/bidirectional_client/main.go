package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Request struct {
	ID      string `json:"id"`
	Method  string `json:"method"`
	Message string `json:"message"`
}

type Response struct {
	ID     string `json:"id"`
	Result string `json:"result"`
}

type ClientIDMessage struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func main() {
	log.Println("🔗 Starting bidirectional SSE client...")

	// Connect to SSE stream
	req, err := http.NewRequest("GET", "http://localhost:8082/events?client_id=demo_client", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{
		Timeout: 0, // No timeout for SSE connection
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server returned status: %d", resp.StatusCode)
	}

	log.Printf("✅ Connected to SSE stream! Status: %d", resp.StatusCode)

	var clientID string
	
	// Read SSE stream and handle requests
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE data
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			log.Printf("📨 Received SSE data: %s", data)

			// Handle client ID message
			var clientIDMsg ClientIDMessage
			if json.Unmarshal([]byte(data), &clientIDMsg) == nil && clientIDMsg.Type == "client_id" {
				clientID = clientIDMsg.ID
				log.Printf("🆔 Got client ID: %s", clientID)
				continue
			}

			// Handle request message
			var request Request
			if err := json.Unmarshal([]byte(data), &request); err != nil {
				log.Printf("❌ Failed to parse request: %v", err)
				continue
			}

			log.Printf("📥 Received request: ID=%s, Method=%s, Message=%s", 
				request.ID, request.Method, request.Message)

			// Process the request (simulate work)
			result := processRequest(request)

			// Send response back via HTTP POST
			response := Response{
				ID:     request.ID,
				Result: result,
			}

			if err := sendResponse(clientID, response); err != nil {
				log.Printf("❌ Failed to send response: %v", err)
			} else {
				log.Printf("📤 Sent response: %s", result)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("❌ SSE connection error: %v", err)
	}

	log.Println("🔚 SSE connection closed")
}

func processRequest(req Request) string {
	// Simulate processing the request (like calling an LLM)
	log.Printf("🔄 Processing request: %s", req.Message)
	
	// Simulate some work
	time.Sleep(1 * time.Second)
	
	// Generate a mock response
	switch req.Method {
	case "analyze":
		return fmt.Sprintf("Analysis result for '%s': This message contains %d characters and appears to be a %s request.", 
			req.Message, len(req.Message), req.Method)
	default:
		return fmt.Sprintf("Processed '%s' using method '%s'", req.Message, req.Method)
	}
}

func sendResponse(clientID string, response Response) error {
	// Convert response to JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %v", err)
	}

	// Create HTTP POST request
	req, err := http.NewRequest("POST", "http://localhost:8082/response", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-ID", clientID)

	// Send the response
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}