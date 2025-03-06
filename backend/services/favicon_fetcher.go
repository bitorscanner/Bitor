package services

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FaviconFetcher handles fetching favicons from websites
type FaviconFetcher struct {
	client *http.Client
}

// NewFaviconFetcher creates a new instance of FaviconFetcher
func NewFaviconFetcher() *FaviconFetcher {
	// Create a custom HTTP client with reasonable timeouts and TLS config
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return &FaviconFetcher{
		client: client,
	}
}

// FetchFavicon attempts to fetch a favicon from a domain using multiple methods
func (f *FaviconFetcher) FetchFavicon(domain string) ([]byte, string, error) {
	// Clean and normalize the domain
	domain = strings.TrimSpace(domain)
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}

	parsedURL, err := url.Parse(domain)
	if err != nil {
		return nil, "", fmt.Errorf("invalid domain: %v", err)
	}

	// List of potential favicon locations to try
	locations := []string{
		"/favicon.ico",
		"/favicon.png",
		"/apple-touch-icon.png",
		"/apple-touch-icon-precomposed.png",
	}

	// Try to get favicon from HTML first
	if faviconData, contentType, err := f.getFaviconFromHTML(parsedURL.String()); err == nil {
		return faviconData, contentType, nil
	}

	// Try each location in order
	for _, location := range locations {
		faviconURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, location)
		if faviconData, contentType, err := f.fetchURL(faviconURL); err == nil {
			return faviconData, contentType, nil
		}
	}

	// Try Google Favicon service as a last resort
	googleFaviconURL := fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=64", parsedURL.Host)
	if faviconData, contentType, err := f.fetchURL(googleFaviconURL); err == nil {
		return faviconData, contentType, nil
	}

	return nil, "", fmt.Errorf("no favicon found for domain %s", domain)
}

// getFaviconFromHTML attempts to extract favicon URL from HTML head
func (f *FaviconFetcher) getFaviconFromHTML(urlStr string) ([]byte, string, error) {
	resp, err := f.client.Get(urlStr)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Convert body to lowercase for easier searching
	htmlStr := strings.ToLower(string(body))

	// List of patterns to search for favicon in HTML
	patterns := []string{
		`<link rel="icon" href="`,
		`<link rel="shortcut icon" href="`,
		`<link rel="apple-touch-icon" href="`,
		`<link rel="apple-touch-icon-precomposed" href="`,
	}

	for _, pattern := range patterns {
		if idx := strings.Index(htmlStr, pattern); idx != -1 {
			// Extract the URL
			start := idx + len(pattern)
			end := strings.Index(htmlStr[start:], `"`)
			if end == -1 {
				continue
			}
			faviconURL := htmlStr[start : start+end]

			// Handle relative URLs
			if !strings.HasPrefix(faviconURL, "http") {
				baseURL, err := url.Parse(urlStr)
				if err != nil {
					continue
				}
				if strings.HasPrefix(faviconURL, "//") {
					faviconURL = baseURL.Scheme + ":" + faviconURL
				} else if strings.HasPrefix(faviconURL, "/") {
					faviconURL = fmt.Sprintf("%s://%s%s", baseURL.Scheme, baseURL.Host, faviconURL)
				} else {
					faviconURL = fmt.Sprintf("%s://%s/%s", baseURL.Scheme, baseURL.Host, faviconURL)
				}
			}

			// Fetch the favicon from the found URL
			return f.fetchURL(faviconURL)
		}
	}

	return nil, "", fmt.Errorf("no favicon found in HTML")
}

// fetchURL fetches data from a URL and returns the data and content type
func (f *FaviconFetcher) fetchURL(url string) ([]byte, string, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Verify the content is an image
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", fmt.Errorf("not an image: %s", contentType)
	}

	return data, contentType, nil
}

// GetDataURI converts image data to a data URI
func (f *FaviconFetcher) GetDataURI(data []byte, contentType string) string {
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data))
}
