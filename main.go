package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	// "time"
)

// Structure to match your extension's data format
type RequestData struct {
	URL           string   `json:"url"`
	LoginDetected bool     `json:"loginDetected"`
	PolicyLinks   []string `json:"policyLinks"`
}

// Stack to store unique RequestData entries
type Stack struct {
	items []RequestData
}

var requestStack Stack
var nonEmptyPolicyStack Stack // New stack for items with non-empty policy links
var seenPolicies = make(map[string]bool)

func (s *Stack) Push(data RequestData) {
	s.items = append(s.items, data)
}

// getPolicyKey creates a unique key from sorted policy links
func getPolicyKey(policies []string) string {
	// Sort policies to ensure consistent comparison
	sortedPolicies := make([]string, len(policies))
	copy(sortedPolicies, policies)
	sort.Strings(sortedPolicies)
	dataBytes, _ := json.Marshal(sortedPolicies)
	return string(dataBytes)
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

		// Print every received request for monitoring with timestamp
		// fmt.Println("\n=== Received Data ===")
		// fmt.Printf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05 -0700"))
		// fmt.Printf("URL: %s\n", data.URL)
		// fmt.Printf("Has Login Form: %v\n", data.LoginDetected)
		// fmt.Println("Policy Links:")
		// for _, link := range data.PolicyLinks {
		// 	fmt.Printf("- %s\n", link)
		// }

		// Create a unique key based on policy links
		policyKey := getPolicyKey(data.PolicyLinks)

		// Check for duplicate policies
		if seenPolicies[policyKey] {
			fmt.Println("Status: Duplicate policies ignored")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "duplicate policies ignored"})
			return
		}

		// Mark policies as seen and push to stack
		seenPolicies[policyKey] = true
		requestStack.Push(data)

		// Push to nonEmptyPolicyStack if PolicyLinks is not empty
		if len(data.PolicyLinks) > 0 {
			nonEmptyPolicyStack.Push(data)

			// Print nonEmptyPolicyStack contents
			fmt.Println("\n=== Non-Empty Policy Stack ===")
			for i, item := range nonEmptyPolicyStack.items {
				fmt.Printf("Item %d:\n", i+1)
				fmt.Printf("  URL: %s\n", item.URL)
				fmt.Printf("  Has Login Form: %v\n", item.LoginDetected)
				fmt.Println("  Policy Links:")
				for _, link := range item.PolicyLinks {
					fmt.Printf("    - %s\n", link)
				}
			}
		}

		// Print current stack contents
		fmt.Println("\n=== Current Stack ===")
		for i, item := range requestStack.items {
			fmt.Printf("Item %d:\n", i+1)
			fmt.Printf("  URL: %s\n", item.URL)
			fmt.Printf("  Has Login Form: %v\n", item.LoginDetected)
			fmt.Println("  Policy Links:")
			for _, link := range item.PolicyLinks {
				fmt.Printf("    - %s\n", link)
			}
		}

		// Send simple response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "received"})
	})

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
