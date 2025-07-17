package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
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
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
		return
	}

	var data RequestData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	policyKey := getPolicyKey(data.PolicyLinks)
	if seenPolicies[policyKey] {
		fmt.Println("Status: Duplicate policies ignored")
		c.JSON(http.StatusOK, gin.H{"status": "duplicate policies ignored"})
		return
	}

	seenPolicies[policyKey] = true
	requestStack.Push(data)

	if len(data.PolicyLinks) > 0 {
		NonEmptyPolicyStack.Push(data)
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

		ScrapeFromStack(&NonEmptyPolicyStack)

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

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

// ScrapeFromStack processes nonEmptyPolicyStack and updates TextPolicy
func ScrapeFromStack(stack *Stack) {
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
		itemsToProcess[i].TextPolicy = policyContent.String()
		itemsToProcess[i].IsProcessed = true
		stack.Push(itemsToProcess[i])
	}
}

// fetchAndStorePolicy scrapes a policy page and returns cleaned, relevant content
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

	var policyContent strings.Builder

	// Target policy-related DOM elements
	c.OnHTML("p, section, article, div:not([class*='nav'],[class*='footer'],[class*='menu'],[class*='banner'],[class*='signup'],[id*='nav'],[id*='footer'],[id*='menu'],[id*='banner'],[id*='signup'])", func(e *colly.HTMLElement) {
		// Skip elements with unwanted attributes or text
		class := strings.ToLower(e.Attr("class"))
		id := strings.ToLower(e.Attr("id"))
		text := strings.ToLower(e.Text)
		if strings.Contains(class, "cookie-consent") || strings.Contains(id, "cookie-consent") ||
			strings.Contains(text, "create an account") || strings.Contains(text, "sign up") ||
			strings.Contains(text, "back to top") || strings.Contains(text, "equal opportunity") ||
			strings.Contains(text, "cookie preferences") || strings.Contains(text, "socialitems") ||
			strings.Contains(text, "facebook") || strings.Contains(text, "linkedin") ||
			strings.Contains(text, "twitter") || strings.Contains(text, "instagram") ||
			strings.Contains(text, "--rg-gradient") || strings.Contains(text, "data-eb-") ||
			strings.Contains(text, "contact us") || strings.Contains(text, "support ticket") ||
			strings.Contains(text, "accessibility") {
			return
		}

		// Prioritize elements with policy-related keywords
		policyKeywords := []string{
			"personal information", "data collection", "third party", "third-party",
			"privacy", "policy", "terms", "data", "cookies", "legal", "retention",
			"security", "access", "children", "location of", "use personal",
			"share personal", "data privacy", "information collected",
		}
		hasPolicyContent := false
		for _, keyword := range policyKeywords {
			if strings.Contains(text, keyword) {
				hasPolicyContent = true
				break
			}
		}

		if hasPolicyContent {
			// Clean the text
			cleanedText := strings.TrimSpace(e.Text)
			if cleanedText == "" {
				return
			}

			// Remove excessive whitespace, special characters, and unwanted patterns
			cleanedText = strings.ReplaceAll(cleanedText, "\n", " ")
			cleanedText = strings.ReplaceAll(cleanedText, "\t", " ")
			cleanedText = regexp.MustCompile(`\s+`).ReplaceAllString(cleanedText, " ")
			cleanedText = regexp.MustCompile(`--rg-gradient[^}]*}`).ReplaceAllString(cleanedText, "")
			cleanedText = regexp.MustCompile(`\[data-eb-[^\]]*\]`).ReplaceAllString(cleanedText, "")
			cleanedText = regexp.MustCompile(`\{[\s\S]*\}`).ReplaceAllString(cleanedText, "") // Remove JSON-like content

			// Only include text with reasonable length
			if len(cleanedText) > 20 {
				policyContent.WriteString(cleanedText)
				policyContent.WriteString("\n")
			}
		}
	})

	// Explicitly ignore non-content elements
	c.OnHTML("script, style, noscript, iframe, footer, header, nav, [class*='cookie-consent'], [class*='signup'], [id*='cookie-consent'], [id*='signup']", func(e *colly.HTMLElement) {
		// Ignore these elements
	})

	// Log the scraping process
	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Scraping policy page: %s\n", r.URL.String())
	})

	err = c.Visit(policyURL)
	if err != nil {
		fmt.Println("Failed to visit policy page:", err)
		return ""
	}

	content := policyContent.String()
	if content == "" {
		fmt.Println("No relevant policy content found for:", policyURL)
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
	r := gin.Default()
	r.POST("/monitor", MainHandler)
	log.Println("Server starting...")

	// Try port 8080 first
	port := ":8080"
	err := r.Run(port)
	if err != nil {
		log.Printf("Port 8080 busy or failed: %v", err)
		log.Println("Trying port 8081...")
		port = ":8081"
		err = r.Run(port)
		if err != nil {
			log.Fatalf("Failed to start server on 8081: %v", err)
		}
	}
}
