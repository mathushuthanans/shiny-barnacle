package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gocolly/colly"
)

func main() {
	startURL := "https://www.youtube.com"

	u, err := url.Parse(startURL)
	if err != nil {
		fmt.Println("Invalid start URL:", err)
		return
	}
	host := u.Hostname()
	wwwHost := "www." + strings.TrimPrefix(host, "www.")

	c := colly.NewCollector(
		colly.AllowedDomains(host, wwwHost),
	)

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		lowerHref := strings.ToLower(href)

		if strings.Contains(lowerHref, "terms") || strings.Contains(lowerHref, "policy") || strings.Contains(lowerHref, "privacy") {
			link := e.Request.AbsoluteURL(href)
			fmt.Println("Found link:", link)
			fetchAndPrintPolicy(link)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting:", r.URL.String())
	})

	err = c.Visit(startURL)
	if err != nil {
		fmt.Println("Failed to visit:", err)
	}
}

func fetchAndPrintPolicy(link string) {
	u, err := url.Parse(link)
	if err != nil {
		fmt.Println("Invalid policy link:", err)
		return
	}

	policyCollector := colly.NewCollector(
		colly.AllowedDomains(u.Hostname(), "www."+strings.TrimPrefix(u.Hostname(), "www.")),
	)

	policyCollector.OnHTML("body", func(e *colly.HTMLElement) {
		fmt.Println("----- Start of Policy Content -----")
		fmt.Println(strings.TrimSpace(e.Text))
		fmt.Println("----- End of Policy Content -----")
	})

	err = policyCollector.Visit(link)
	if err != nil {
		fmt.Println("Failed to visit policy page:", err)
	}
}
