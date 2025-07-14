package services

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gocolly/colly"
)

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
