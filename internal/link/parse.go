package link

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func Parse(htmlBody, rawBaseURL string) ([]string, error) {
	reader := strings.NewReader(htmlBody)
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %s", err)
	}

	nodes := linkNodes(doc)

	var links []string
	for _, node := range nodes {
		href := findHref(node)
		if href == "" {
			continue
		}

		fullURL, err := makeFullURL(rawBaseURL, href)
		if err != nil {
			continue
		}

		links = append(links, fullURL.String())
	}

	return links, nil
}
func findHref(n *html.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			return attr.Val
		}
	}

	return ""
}

func linkNodes(n *html.Node) []*html.Node {
	if n.Type == html.ElementNode && n.Data == "a" {
		return []*html.Node{n}
	}

	var ret []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ret = append(ret, linkNodes(c)...)
	}

	return ret
}

func makeFullURL(rawBaseURL, link string) (*url.URL, error) {
	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %s", err)
	}

	u, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %s", err)
	}

	if u.IsAbs() {
		return u, nil
	}

	return baseURL.ResolveReference(u), nil
}
