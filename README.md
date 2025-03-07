# webscraper

a fast, concurrent dead link checker built with Go and Playwright. it crawls a given URL, detects broken pages, and generates a clean report.

## features

- **concurrent scraping**: leverages go routines and a semaphore for max concurrency.
- **smart crawling**: only digs into pages on the same host.
- **detailed reports**: optionally shows why a link is dead.

## requirements

- go (version 1.23+)
- [playwright-go](https://github.com/playwright-community/playwright-go)

## installation

1. **install playwright-go**: follow the installation guide in the [playwright-go readme](https://github.com/playwright-community/playwright-go/blob/main/README.md#installation).

2. **clone this repo**

3. **run the scraper**:
```bash
go run main.go -url https://example.com -max-concurrency 5 -max-pages 0 -print-reason -timeout 15s
```

adjust flags as needed.

## usage

```
Usage: webscraper [OPTIONS]

Options:
  -url <URL>                   base url to start scraping (required)
  -max-concurrency <N>         maximum number of concurrent requests (default: 5)
  -max-pages <N>               maximum pages to crawl (0 = unlimited)
  -print-reason                print reason for dead links in the report
  -timeout <DURATION>          timeout for each request (e.g., 15s, 1m) (default: 15s)
  -h, --help                   print help
```

## notes

- the tool uses playwright for browser automation, so ensure your environment supports it.
- only pages from the same host are crawled—keeps things tidy.

enjoy and happy scraping, fam! 
