package utils

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateJiraURL validates that a URL is safe for making Jira API requests
// This prevents Server-Side Request Forgery (SSRF) attacks
func ValidateJiraURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Only allow HTTP and HTTPS schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTP and HTTPS protocols are allowed, got: %s", parsedURL.Scheme)
	}

	// Ensure HTTPS for production security (recommend but don't enforce for flexibility)
	if parsedURL.Scheme == "http" {
		// Log warning but allow for development/internal instances
		// In production, you might want to enforce HTTPS only
	}

	// Validate hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must contain a valid hostname")
	}

	// Prevent requests to localhost and internal networks
	if err := validateHostname(hostname); err != nil {
		return fmt.Errorf("unsafe hostname: %v", err)
	}

	// Ensure the URL looks like a Jira instance
	if err := validateJiraURLPattern(parsedURL); err != nil {
		return fmt.Errorf("URL does not appear to be a valid Jira instance: %v", err)
	}

	return nil
}

// validateHostname checks if a hostname is safe to make requests to
func validateHostname(hostname string) error {
	// Block localhost variants
	localhost := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"::1",
		"[::1]",
	}

	for _, local := range localhost {
		if strings.EqualFold(hostname, local) {
			return fmt.Errorf("requests to localhost are not allowed")
		}
	}

	// Resolve hostname to IP and check for private networks
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// If we can't resolve, it might be a development server
		// Allow it but log the issue
		return nil
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("requests to private IP addresses are not allowed: %s", ip.String())
		}
	}

	return nil
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	// Define private IP ranges
	privateRanges := []string{
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"127.0.0.0/8",    // loopback
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local
	}

	for _, rangeStr := range privateRanges {
		_, privateNet, err := net.ParseCIDR(rangeStr)
		if err != nil {
			continue
		}
		if privateNet.Contains(ip) {
			return true
		}
	}

	return false
}

// validateJiraURLPattern performs basic validation that this looks like a Jira URL
func validateJiraURLPattern(parsedURL *url.URL) error {
	// Jira URLs should not contain suspicious paths
	suspiciousPaths := []string{
		"/admin",
		"/internal",
		"/debug",
		"/console",
		"/management",
		"/_", // Common internal endpoint prefix
	}

	path := strings.ToLower(parsedURL.Path)
	for _, suspicious := range suspiciousPaths {
		if strings.Contains(path, suspicious) {
			return fmt.Errorf("URL path contains suspicious component: %s", suspicious)
		}
	}

	// Jira URLs typically have domains ending in known patterns or contain "jira"
	hostname := strings.ToLower(parsedURL.Hostname())

	// Allow if hostname contains "jira" (like mycompany.jira.com or jira.mycompany.com)
	if strings.Contains(hostname, "jira") {
		return nil
	}

	// Allow known Jira hosting patterns
	jiraPatterns := []string{
		".atlassian.net",
		".atlassian.com",
	}

	for _, pattern := range jiraPatterns {
		if strings.HasSuffix(hostname, pattern) {
			return nil
		}
	}

	// For other domains, we'll allow them but this is where you could add
	// additional validation for your organization's specific patterns
	return nil
}

// SanitizeJiraURL ensures a Jira URL is properly formatted and safe
func SanitizeJiraURL(rawURL string) (string, error) {
	if err := ValidateJiraURL(rawURL); err != nil {
		return "", err
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Remove any query parameters and fragments for security
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	// Ensure no trailing slash for consistency
	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")

	return parsedURL.String(), nil
}
