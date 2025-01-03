package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/shadycyan/webscraper/internal/link"
	"github.com/shadycyan/webscraper/internal/safemap"
)

type config struct {
	baseURL *url.URL
	pages   *safemap.SafeMap[string, page]
	wg      *sync.WaitGroup
	sem     chan struct{}
}

type page struct {
	url       string
	sourceURL string
	isDead    bool
	reason    string
}

const (
	httpTimeout  = 5 * time.Second
	expectedType = "text/html"
	greenColor   = "\033[32m"
	resetColor   = "\033[0m"
)

func main() {
	baseURL := flag.String("url", "", "URL to scrape")
	maxRequests := flag.Int("max-requests", 10, "Maximum number of concurrent requests")
	flag.Parse()

	if *baseURL == "" {
		fmt.Println("please provide a URL to scrape using the -url flag")
		return
	}

	parsedBaseURL, err := url.Parse(*baseURL)
	if err != nil {
		fmt.Printf("failed to parse base URL: %s\n", err)
		return
	}

	config := config{
		baseURL: parsedBaseURL,
		pages:   safemap.New[string, page](),
		wg:      &sync.WaitGroup{},
		sem:     make(chan struct{}, *maxRequests),
	}

	config.wg.Add(1)
	go func() {
		defer config.wg.Done()
		config.processPage(*baseURL, *baseURL)
	}()

	config.wg.Wait()

	deadLinks := slices.DeleteFunc(
		config.pages.Values(),
		func(p page) bool { return p.isDead == false },
	)

	printReport(deadLinks)
}

func (cfg *config) processPage(rawCurrentURL, sourceURL string) {
	fmt.Println("checking", rawCurrentURL)

	normalizedURL, uri, err := link.NormalizeURL(rawCurrentURL)
	if err != nil {
		return
	}

	if cfg.pages.Contains(normalizedURL) {
		fmt.Println("skipping", rawCurrentURL)
		return
	}

	p := page{url: rawCurrentURL, sourceURL: sourceURL}

	html, err := cfg.readPage(rawCurrentURL)
	if err != nil {
		var contentTypeErr *contentTypeError
		if !errors.As(err, &contentTypeErr) {
			p.isDead = true
			p.reason = err.Error()
		}

		cfg.pages.Set(normalizedURL, p)
		return
	}

	cfg.pages.Set(normalizedURL, p)

	isSame := cfg.baseURL.Host == uri.Host
	if !isSame {
		return
	}

	links, err := link.Parse(html, rawCurrentURL)

	fmt.Printf("found %v\n", links)

	for _, link := range links {
		cfg.wg.Add(1)

		go func() {
			defer cfg.wg.Done()
			cfg.processPage(link, rawCurrentURL)
		}()
	}
}

func (cfg *config) readPage(rawURL string) (string, error) {
	client := &http.Client{Timeout: httpTimeout}

	cfg.sem <- struct{}{}
	defer func() { <-cfg.sem }()

	resp, err := client.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", &statusCodeError{statusCode: resp.StatusCode, statusText: http.StatusText(resp.StatusCode)}
	}

	contentType := resp.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", fmt.Errorf("failed to parse media type: %w", err)
	}

	if mediaType != expectedType {
		return "", &contentTypeError{contentType: contentType, expectedType: expectedType}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func printReport(pages []page) {
	if len(pages) == 0 {
		fmt.Println("did't find any dead links")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "%s\n", greenColor)

	fmt.Fprintf(w, "Page\tLink\n")

	for _, page := range pages {
		fmt.Fprintf(w, "%s\t%s\n", page.sourceURL, page.url)
	}

	fmt.Fprintf(w, "%s\n", resetColor)
}
