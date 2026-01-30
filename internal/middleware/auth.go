package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rizkysr90/aslam-flower/internal/services"
)

// AuthRequired middleware verifies JWT token from cookie
// If valid, sets user_id and username in locals
// If invalid, redirects to /admin/login
func AuthRequired(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from cookie
		token := c.Cookies("auth_token")
		if token == "" {
			return c.Redirect("/admin/login")
		}

		// Verify token
		claims, err := authService.VerifyToken(token)
		if err != nil {
			// Clear invalid/expired token
			c.ClearCookie("auth_token")
			return c.Redirect("/admin/login")
		}

		// Store user info in context
		c.Locals("user_id", claims.UserID)
		c.Locals("username", claims.Username)

		return c.Next()
	}
}
