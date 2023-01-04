package main

// Features find all the links tagged with  A hrefs
// Calculate Pages per second
// Find links async

import (
	"encoding/json"
	"fmt"
	"github.com/bobesa/go-domain-util/domainutil"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var requestCount = 0
var startTime = time.Now()

// ScanConfig represents the configuration data from the JSON file
type ScanConfig struct {
	RootDomainScope bool `json:"rootDomainScope"`
	CrawlDepth      int  `json:"crawlDepth"`
	Timeout         int  `json:"timeout"`
}

// PagesPerSecond takes an integer as an input and calculates divides the integer by execution time
func PagesPerSecond(n int, startTime time.Time) float64 {

	var sum int
	for i := 1; i <= n; i++ {
		sum += i
	}

	elapsedTime := time.Since(startTime)
	intsPerSecond := float64(n) / elapsedTime.Seconds()

	fmt.Printf("Processed %d Page in %v\n", n, elapsedTime)
	fmt.Printf("%.2f Pages per second\n", intsPerSecond)

	return intsPerSecond
}

func getConfig() ScanConfig {
	// Open the config file
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			println(err)
		}
	}(file)

	// Decode the JSON data into a Config struct
	var config ScanConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		fmt.Println(err)

	}

	// Use the config data
	fmt.Println("RootDomainScope:", config.RootDomainScope)
	fmt.Println("CrawlDepth:", config.CrawlDepth)
	fmt.Println("Timeout:", config.Timeout)
	return config

}

var proxies []*url.URL = []*url.URL{
	&url.URL{Host: "127.0.0.1:8080"},
	&url.URL{Host: "127.0.0.1:8081"},
}

func randomProxySwitcher(_ *http.Request) (*url.URL, error) {
	return proxies[rand.Intn(len(proxies))], nil
}

func allSubdomains(target string) []*regexp.Regexp {
	rootDomain := domainutil.Domain(target)
	pattern := "^https?:\\/\\/(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\\.)+" + rootDomain
	regexps := []*regexp.Regexp{regexp.MustCompile(pattern)}
	return regexps

}

func linkValidation(url string) string {

	if strings.HasPrefix(url, "mailto:") != true {

	}
	return url
}

func crawl(target string) {
	config := getConfig()
	// Create a new collector

	c := colly.NewCollector(
		colly.Async(true),                 // Enable asynchronous crawling
		colly.MaxDepth(config.CrawlDepth), // Set the maximum crawl depth
		// Attach a debugger to the collector
		//colly.Debugger(&debug.LogDebugger{}),
		colly.IgnoreRobotsTxt(),
		colly.ParseHTTPErrorResponse(),
		colly.URLFilters(allSubdomains(target)...),
	)

	sessionCookie := http.Cookie{
		Name:  "region_id",
		Value: "2",
		Path:  "/",
	}
	var cookies []*http.Cookie
	cookies = append(cookies, &sessionCookie)

	c.SetCookies(domainutil.Domain(target), cookies)
	
	//c.SetProxyFunc(randomProxySwitcher)
	// Limit the number of threads started by colly to two
	// when visiting links which domains' matches "*httpbin.*" glob
	c.Limit(&colly.LimitRule{
		Parallelism: 4,
		RandomDelay: 5 * time.Second,
	})
	// Increase the timeout
	c.SetRequestTimeout(time.Duration(config.Timeout) * time.Second)
	extensions.RandomUserAgent(c)

	// Set up an onResponse callback
	c.OnResponse(func(r *colly.Response) {
		// Print the response status code and URL
		requestCount++
		fmt.Println("Found:", r.Request.URL, r.StatusCode)
	})

	// Set up a handler for links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// Find all A hrefs and put them back in the stack
		link := e.Attr("href")

		// Check for known bad patterns
		link = linkValidation(link)

		if strings.HasPrefix(link, "mailto:") != true {
			//err := q.AddURL(e.Request.AbsoluteURL(link))
			err := e.Request.Visit(link)
			if err != nil {
				return
			}

		}

	})

	// Set up a handler for errors
	c.OnError(func(r *colly.Response, err error) {

		//fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, r.StatusCode, "\nError:", err)
	})

	// Start scraping the website
	err := c.Visit(target)
	if err != nil {
		log.Fatal(err)
	}

	// Wait for all scraping jobs to complete
	c.Wait()

	fmt.Printf("\nFinished Crawling\n")
	PagesPerSecond(requestCount, startTime)
}

func main() {
	target := "https://www.target.com"
	crawl(target)

}

