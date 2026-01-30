package handlers

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/rizkysr90/aslam-flower/internal/services"
)

// AuthHandler handles authentication routes
type AuthHandler struct {
	authService *services.AuthService
	env         string
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		env:         os.Getenv("ENV"),
	}
}

// LoginPage renders the login form
func (h *AuthHandler) LoginPage(c *fiber.Ctx) error {
	// Check if already logged in
	token := c.Cookies("auth_token")
	if token != "" {
		// Verify token
		_, err := h.authService.VerifyToken(token)
		if err == nil {
			// Already authenticated, redirect to dashboard
			return c.Redirect("/admin/dashboard")
		}
		// Invalid token, clear it
		c.ClearCookie("auth_token")
	}

	// Get error message from query if present
	errorMsg := c.Query("error", "")

	// Get CSRF token from context (set by CSRF middleware) or cookie
	csrfToken := ""
	if token := c.Locals("csrf_token"); token != nil {
		csrfToken = token.(string)
	} else {
		csrfToken = c.Cookies("csrf_")
	}

	return c.Render("pages/admin/login", fiber.Map{
		"Title":     "Admin Login",
		"Error":     errorMsg,
		"Username":  c.Query("username", ""), // Pre-fill username if provided
		"CSRFToken": csrfToken,
	})
}

// Login handles login form submission
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	// Parse form data
	username := c.FormValue("username")
	password := c.FormValue("password")

	// Validate input
	if username == "" || password == "" {
		csrfToken := ""
		if token := c.Locals("csrf_token"); token != nil {
			csrfToken = token.(string)
		} else {
			csrfToken = c.Cookies("csrf_")
		}

		return c.Render("pages/admin/login", fiber.Map{
			"Title":     "Admin Login",
			"Error":     "Username and password are required",
			"Username":  username,
			"CSRFToken": csrfToken,
		})
	}

	// Attempt login
	token, err := h.authService.Login(username, password)
	if err != nil {
		// Login failed - show error
		csrfToken := ""
		if token := c.Locals("csrf_token"); token != nil {
			csrfToken = token.(string)
		} else {
			csrfToken = c.Cookies("csrf_")
		}

		return c.Render("pages/admin/login", fiber.Map{
			"Title":     "Admin Login",
			"Error":     "Invalid username or password",
			"Username":  username, // Keep username for retry
			"CSRFToken": csrfToken,
		})
	}

	// Login successful - set cookie
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400,                 // 24 hours in seconds
		HTTPOnly: true,                  // Prevent XSS access
		Secure:   h.env == "production", // HTTPS only in production
		SameSite: "Strict",              // CSRF protection
	})

	// Redirect to dashboard
	return c.Redirect("/admin/dashboard")
}

// Logout handles logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// Clear auth cookie
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete cookie
		HTTPOnly: true,
		Secure:   h.env == "production",
		SameSite: "Strict",
	})

	// Redirect to login page
	return c.Redirect("/admin/login")
}
