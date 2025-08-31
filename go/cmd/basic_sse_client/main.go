package main

import (
	"bufio"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	log.Println("🔗 Connecting to SSE server...")

	// Create HTTP request for SSE endpoint
	req, err := http.NewRequest("GET", "http://localhost:8081/events", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second, // Total request timeout
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server returned status: %d", resp.StatusCode)
	}

	log.Printf("✅ Connected! Status: %d", resp.StatusCode)
	log.Println("📡 Listening for events...")

	// Read SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse SSE event format
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			log.Printf("📨 Received: %s", data)
		} else {
			log.Printf("🔍 Other line: %s", line)
		}
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		log.Printf("❌ Connection error: %v", err)
	}

	log.Println("🔚 Connection closed")
}