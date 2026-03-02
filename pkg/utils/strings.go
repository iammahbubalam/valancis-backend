package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// GenerateSlug converts a string into a URL-friendly slug.
// e.g. "Men's T-Shirt!" -> "mens-t-shirt"
func GenerateSlug(input string) string {
	// Convert to lower case
	s := strings.ToLower(input)

	// Remove invalid chars (keep a-z, 0-9, space, hyphen)
	reg := regexp.MustCompile("[^a-z0-9 -]+")
	s = reg.ReplaceAllString(s, "")

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Collapse multiple hyphens
	reg2 := regexp.MustCompile("-+")
	s = reg2.ReplaceAllString(s, "-")

	// Trim hyphens
	s = strings.Trim(s, "-")

	return s
}

// ParseInt parses a string to int with a fallback default value
func ParseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	// We use Atoi which is equivalent to ParseInt(s, 10, 0) converted to int
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}
