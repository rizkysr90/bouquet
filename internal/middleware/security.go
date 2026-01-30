package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders middleware sets security headers
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Prevent MIME type sniffing
		c.Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Set("X-Frame-Options", "DENY")

		// XSS protection
		c.Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		// Allow self, unpkg for htmx, Tailwind CDN, Cloudinary for images
		// Note: 'unsafe-inline' is required for inline scripts in templates (e.g., product-detail.html)
		// Note: data: is allowed for inline SVG placeholders
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.tailwindcss.com; " +
			"img-src 'self' https://res.cloudinary.com data:; " +
			"style-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com; " +
			"font-src 'self' data:; " +
			"connect-src 'self';"
		c.Set("Content-Security-Policy", csp)

		// Strict Transport Security (HTTPS only in production)
		if os.Getenv("ENV") == "production" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		return c.Next()
	}
}
