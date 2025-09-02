package tailscale

import (
	"regexp"
	"strings"
)

// Regular expression to match IPv4 addresses
var ipv4Regex = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)

// Regular expression to match IPv6 addresses (simplified pattern)
var ipv6Regex = regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b|\b(?:[0-9a-fA-F]{1,4}:){1,7}:\b|\b(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}\b|\b(?:[0-9a-fA-F]{1,4}:){1,5}(?::[0-9a-fA-F]{1,4}){1,2}\b|\b(?:[0-9a-fA-F]{1,4}:){1,4}(?::[0-9a-fA-F]{1,4}){1,3}\b|\b(?:[0-9a-fA-F]{1,4}:){1,3}(?::[0-9a-fA-F]{1,4}){1,4}\b|\b(?:[0-9a-fA-F]{1,4}:){1,2}(?::[0-9a-fA-F]{1,4}){1,5}\b|\b[0-9a-fA-F]{1,4}:(?::[0-9a-fA-F]{1,4}){1,6}\b|\b:(?:(?::[0-9a-fA-F]{1,4}){1,7}|:)\b`)

// ExtractIPAddresses extracts all IP addresses (both IPv4 and IPv6) from a message
func ExtractIPAddresses(message string) []string {
	ipAddresses := make([]string, 0)

	// Extract IPv4 addresses
	ipv4Matches := ipv4Regex.FindAllString(message, -1)
	for _, match := range ipv4Matches {
		if !contains(ipAddresses, match) {
			ipAddresses = append(ipAddresses, match)
		}
	}

	// Extract IPv6 addresses
	ipv6Matches := ipv6Regex.FindAllString(message, -1)
	for _, match := range ipv6Matches {
		if !contains(ipAddresses, match) {
			ipAddresses = append(ipAddresses, match)
		}
	}

	return ipAddresses
}

// contains checks if a string is present in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}