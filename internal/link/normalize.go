package link

import (
	"fmt"
	"net/url"
	"strings"
)

func NormalizeURL(uriStr string) (string, url.URL, error) {
	uri, err := url.Parse(uriStr)
	if err != nil {
		return "", url.URL{}, fmt.Errorf("failed to parse uri: %w", err)
	}

	uri = uri.JoinPath()
	uri.Path = strings.TrimSuffix(uri.Path, "/")

	norm := strings.ToLower(uri.Host) + uri.Path

	return norm, *uri, nil
}
