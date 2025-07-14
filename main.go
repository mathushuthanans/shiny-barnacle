package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Structure to match your extension's data format
type RequestData struct {
	URL           string   `json:"url"`
	LoginDetected bool     `json:"loginDetected"`
	PolicyLinks   []string `json:"policyLinks"`
}

func main() {
	http.HandleFunc("/monitor", func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the JSON data
		var data RequestData
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Print to terminal
		fmt.Println("\n=== Received Data ===")
		fmt.Printf("URL: %s\n", data.URL)
		fmt.Printf("Has Login Form: %v\n", data.LoginDetected)
		fmt.Println("Policy Links:")
		for _, link := range data.PolicyLinks {
			fmt.Printf("- %s\n", link)
		}

		// Send simple response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "received"})
	})

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
