package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	// SSE endpoint
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		log.Printf("Client connected: %s", r.RemoteAddr)

		// Send periodic events
		for i := 1; i <= 10; i++ {
			// SSE event format: "data: message\n\n"
			event := fmt.Sprintf("data: Event %d at %s\n\n", i, time.Now().Format("15:04:05"))
			
			log.Printf("Sending: Event %d", i)
			
			// Write event to client
			_, err := fmt.Fprint(w, event)
			if err != nil {
				log.Printf("Client disconnected: %v", err)
				return
			}

			// Flush immediately (important for SSE!)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			// Wait before next event
			time.Sleep(2 * time.Second)
		}

		log.Println("Finished sending events")
	})

	// Serve the HTML test page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/sse-test.html")
	})

	log.Println("🚀 Simple SSE Server starting on :8081")
	log.Println("📱 Open browser: http://localhost:8081")
	log.Println("🔗 SSE endpoint: http://localhost:8081/events")
	log.Fatal(http.ListenAndServe(":8081", nil))
}