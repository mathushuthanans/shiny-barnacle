package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly"
)

// RequestData structure to match the extension's data format
type RequestData struct {
	URL           string   `json:"url"`
	LoginDetected bool     `json:"loginDetected"`
	PolicyLinks   []string `json:"policyLinks"`
	IsProcessed   bool     `json:"isProcessed"`
	TextPolicy    string   `json:"textPolicy"`
}

// Stack to store unique RequestData entries
type Stack struct {
	items []RequestData
}

var requestStack Stack
var NonEmptyPolicyStack Stack
var seenPolicies = make(map[string]bool)

func (s *Stack) Push(data RequestData) {
	s.items = append(s.items, data)
}

// Pop removes and returns the top item from the stack
func (s *Stack) Pop() (RequestData, bool) {
	if len(s.items) == 0 {
		return RequestData{}, false
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item, true
}

// getPolicyKey creates a unique key from sorted policy links
func getPolicyKey(policies []string) string {
	sortedPolicies := make([]string, len(policies))
	copy(sortedPolicies, policies)
	sort.Strings(sortedPolicies)
	dataBytes, _ := json.Marshal(sortedPolicies)
	return string(dataBytes)
}

// MainHandler handles the /monitor endpoint
func MainHandler(c *gin.Context) {
	// Only accept POST requests
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
		return
	}

	// Parse the JSON data
	var data RequestData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	// Create a unique key based on policy links
	policyKey := getPolicyKey(data.PolicyLinks)

	// Check for duplicate policies
	if seenPolicies[policyKey] {
		fmt.Println("Status: Duplicate policies ignored")
		c.JSON(http.StatusOK, gin.H{"status": "duplicate policies ignored"})
		return
	}

	// Mark policies as seen and push to stack
	seenPolicies[policyKey] = true
	requestStack.Push(data)

	// Push to nonEmptyPolicyStack if PolicyLinks is not empty
	if len(data.PolicyLinks) > 0 {
		NonEmptyPolicyStack.Push(data)

		// Print nonEmptyPolicyStack contents before scraping
		fmt.Println("\n=== Non-Empty Policy Stack (Before Scraping) ===")
		for i, item := range NonEmptyPolicyStack.items {
			fmt.Printf("Item %d:\n", i+1)
			fmt.Printf("  URL: %s\n", item.URL)
			fmt.Printf("  Has Login Form: %v\n", item.LoginDetected)
			fmt.Printf("  Is Processed: %v\n", item.IsProcessed)
			fmt.Printf("  Text Policy: %s\n", item.TextPolicy)
			fmt.Println("  Policy Links:")
			for _, link := range item.PolicyLinks {
				fmt.Printf("    - %s\n", link)
			}
		}

		// Scrape and update NonEmptyPolicyStack
		ScrapeFromStack(&NonEmptyPolicyStack)

		// Print updated nonEmptyPolicyStack contents after scraping
		fmt.Println("\n=== Non-Empty Policy Stack (After Scraping) ===")
		for i, item := range NonEmptyPolicyStack.items {
			fmt.Printf("Item %d:\n", i+1)
			fmt.Printf("  URL: %s\n", item.URL)
			fmt.Printf("  Has Login Form: %v\n", item.LoginDetected)
			fmt.Printf("  Is Processed: %v\n", item.IsProcessed)
			fmt.Printf("  Text Policy: %s\n", item.TextPolicy)
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
		fmt.Printf("  Is Processed: %v\n", item.IsProcessed)
		fmt.Printf("  Text Policy: %s\n", item.TextPolicy)
		fmt.Println("  Policy Links:")
		for _, link := range item.PolicyLinks {
			fmt.Printf("    - %s\n", link)
		}
	}

	// Send simple response
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

// ScrapeFromStack processes nonEmptyPolicyStack and updates TextPolicy
func ScrapeFromStack(stack *Stack) {
	// Pop items to process to avoid modifying stack during iteration
	var itemsToProcess []RequestData
	for len(stack.items) > 0 {
		item, ok := stack.Pop()
		if !ok {
			break
		}
		if !item.IsProcessed {
			itemsToProcess = append(itemsToProcess, item)
		}
	}

	// Process each unprocessed item
	for i, item := range itemsToProcess {
		fmt.Printf("Processing stack item: %s\n", item.URL)
		var policyContent strings.Builder
		for _, policyLink := range item.PolicyLinks {
			fmt.Printf("Scraping policy link: %s\n", policyLink)
			content := fetchAndStorePolicy(policyLink)
			if content != "" {
				policyContent.WriteString(strings.TrimSpace(content))
				policyContent.WriteString("\n\n")
			}
		}
		// Update TextPolicy with cumulative content
		itemsToProcess[i].TextPolicy = policyContent.String()
		// Mark item as processed
		itemsToProcess[i].IsProcessed = true
		// Push back to stack
		stack.Push(itemsToProcess[i])
	}
}

// fetchAndStorePolicy scrapes a policy page and returns content
func fetchAndStorePolicy(policyURL string) string {
	u, err := url.Parse(policyURL)
	if err != nil {
		fmt.Println("Invalid policy URL:", err)
		return ""
	}

	host := u.Hostname()
	wwwHost := "www." + strings.TrimPrefix(host, "www.")

	c := colly.NewCollector(
		colly.AllowedDomains(host, wwwHost),
		colly.MaxDepth(2),
	)

	var content string
	c.OnHTML("body", func(e *colly.HTMLElement) {
		content = strings.TrimSpace(e.Text)
	})

	err = c.Visit(policyURL)
	if err != nil {
		fmt.Println("Failed to visit policy page:", err)
		return ""
	}
	return content
}

func scrape() {
	startURL := "https://example.com"

	u, err := url.Parse(startURL)
	if err != nil {
		fmt.Println("Invalid URL:", err)
		return
	}

	host := u.Hostname()
	wwwHost := "www." + strings.TrimPrefix(host, "www.")

	c := colly.NewCollector(
		colly.AllowedDomains(host, wwwHost),
		colly.MaxDepth(2),
	)

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		hrefLower := strings.ToLower(href)

		if strings.Contains(hrefLower, "login") || strings.Contains(hrefLower, "signin") {
			link := e.Request.AbsoluteURL(href)
			fmt.Println("Found login/signin page:", link)
			monitorLoginForPolicy(link)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Scanning homepage:", r.URL.String())
	})

	err = c.Visit(startURL)
	if err != nil {
		fmt.Println("Failed to visit homepage:", err)
	}
}

func monitorLoginForPolicy(loginURL string) {
	u, err := url.Parse(loginURL)
	if err != nil {
		fmt.Println("Invalid login page URL:", err)
		return
	}

	host := u.Hostname()
	wwwHost := "www." + strings.TrimPrefix(host, "www.")

	c := colly.NewCollector(
		colly.AllowedDomains(host, wwwHost),
		colly.MaxDepth(2),
	)

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		hrefLower := strings.ToLower(href)

		if strings.Contains(hrefLower, "terms") || strings.Contains(hrefLower, "policy") || strings.Contains(hrefLower, "privacy") {
			link := e.Request.AbsoluteURL(href)
			fmt.Println("Found policy link on login page:", link)
			fetchAndPrintPolicy(link)
		}
	})

	err = c.Visit(loginURL)
	if err != nil {
		fmt.Println("Failed to visit login page:", err)
	}
}

func fetchAndPrintPolicy(link string) {
	c := colly.NewCollector()

	c.OnHTML("body", func(e *colly.HTMLElement) {
		fmt.Println("----- Start of Policy Content -----")
		fmt.Println(strings.TrimSpace(e.Text))
		fmt.Println("----- End of Policy Content -----")
	})

	err := c.Visit(link)
	if err != nil {
		fmt.Println("Failed to visit policy page:", err)
	}
}

func main() {
	// Initialize Gin router
	r := gin.Default()

	// Define API routes
	r.POST("/monitor", MainHandler)

	// Start server
	log.Println("Server started at :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
