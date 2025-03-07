package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"mime"
	"net/url"
	"os"
	"slices"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/shadycyan/webscraper/internal/link"
	"github.com/shadycyan/webscraper/internal/safemap"
	"golang.org/x/sync/singleflight"
)

type config struct {
	baseURL     *url.URL
	pages       *safemap.SafeMap[string, page]
	wg          *sync.WaitGroup
	sem         chan struct{}
	pw          *playwright.Playwright
	browser     playwright.Browser
	context     playwright.BrowserContext
	printReason bool
	timeout     time.Duration
	maxPages    int
}

type page struct {
	url       string
	sourceURL string
	isDead    bool
	reason    string
}

var requestGroup singleflight.Group

const (
	expectedType = "text/html"
	greenColor   = "\033[32m"
	resetColor   = "\033[0m"
)

func main() {
	baseURL := flag.String("url", "", "Base URL to start scraping from")
	maxConcurrency := flag.Int("max-concurrency", 5, "Maximum number of concurrent requests")
	maxPages := flag.Int("max-pages", 0, "Maximum number of pages to crawl")
	printReason := flag.Bool("print-reason", false, "Print the reason for dead links in the report")
	timeout := flag.Duration("timeout", 15*time.Second, "Timeout duration for each request (e.g., 15s, 1m)")
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

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}
	browser, err := pw.Chromium.Launch()
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.135 Safari/537.36"),
	})
	if err != nil {
		log.Fatalf("could not create context: %v", err)
	}

	config := config{
		baseURL:     parsedBaseURL,
		pages:       safemap.New[string, page](),
		wg:          &sync.WaitGroup{},
		sem:         make(chan struct{}, *maxConcurrency),
		pw:          pw,
		browser:     browser,
		context:     context,
		printReason: *printReason,
		timeout:     *timeout,
		maxPages:    *maxPages,
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

	config.printReport(deadLinks)

	close(config.sem)

	browser.Close()
	pw.Stop()
}

func (cfg *config) processPage(rawCurrentURL, sourceURL string) {
	if cfg.maxPages != 0 && len(cfg.pages.Keys()) > cfg.maxPages {
		return
	}

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

	for _, link := range links {
		cfg.wg.Add(1)

		go func() {
			defer cfg.wg.Done()
			cfg.processPage(link, rawCurrentURL)
		}()
	}
}

type response struct {
	resp playwright.Response
	page playwright.Page
}

func (cfg *config) readPage(rawURL string) (string, error) {
	cfg.sem <- struct{}{}
	defer func() { <-cfg.sem }()

	result, err, _ := requestGroup.Do(rawURL, func() (interface{}, error) {
		page, err := cfg.context.NewPage()
		if err != nil {
			return nil, fmt.Errorf("could not create a new browser page: %v", err)
		}

		resp, err := page.Goto(rawURL, playwright.PageGotoOptions{
			Timeout: playwright.Float(cfg.timeout.Seconds() * 1000),
		})
		if err != nil {
			return nil, fmt.Errorf("request failed for %s: %v", rawURL, err)
		}

		if nil == resp {
			return nil, fmt.Errorf("no response received from %s", rawURL)
		}

		return response{resp: resp, page: page}, nil
	})

	if err != nil {
		return "", fmt.Errorf("error while loading %s: %w", rawURL, err)
	}

	response := result.(response)
	resp := response.resp
	page := response.page
	defer page.Close()

	if resp.Status() >= 400 {
		return "", &statusCodeError{statusCode: resp.Status(), statusText: resp.StatusText()}
	}

	contentType, exists := resp.Headers()["Content-Type"]
	if !exists {
		contentType = resp.Headers()["content-type"]
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", fmt.Errorf("failed to parse media type: %w", err)
	}

	if mediaType != expectedType {
		return "", &contentTypeError{contentType: contentType, expectedType: expectedType}
	}

	body, err := resp.Body()

	return string(body), nil
}

func (cfg *config) printReport(pages []page) {
	if len(pages) == 0 {
		fmt.Println("didn't find any dead links")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	defer func() {
		if err := w.Flush(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to flush writer: %v\n", err)
		}
	}()

	fmt.Fprintf(w, "%s\n", greenColor)

	fmt.Fprintf(w, "Page\tLink")
	if cfg.printReason {
		fmt.Fprintf(w, "\tReason")
	}
	fmt.Fprintf(w, "\n")

	for _, page := range pages {
		fmt.Fprintf(w, "%s\t%s", page.sourceURL, page.url)
		if cfg.printReason {
			fmt.Fprintf(w, "\t%s", page.reason)
		}
		fmt.Fprintf(w, "\n")
	}

	fmt.Fprintf(w, "%s\n", resetColor)
}
