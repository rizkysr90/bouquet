package utils

import (
	"regexp"
	"strings"
)

// GenerateSlug converts a name to a URL-friendly slug
// Example: "Pita & Ribbon" → "pita-ribbon"
func GenerateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	// Match any non-alphanumeric character (except hyphens)
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = re.ReplaceAllString(slug, "-")

	// Remove consecutive hyphens
	re = regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}
