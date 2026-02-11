package middleware

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var loggerWriter io.Writer = os.Stdout

// Logger middleware logs request details in JSON format
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generate request ID
		requestID := uuid.New().String()
		c.Locals("request_id", requestID)

		// Start timer
		start := time.Now()

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Build log entry
		logEntry := map[string]interface{}{
			"request_id":  requestID,
			"method":      c.Method(),
			"path":        c.Path(),
			"status":      c.Response().StatusCode(),
			"duration_ms": duration.Milliseconds(),
			"ip":          c.IP(),
			"user_agent":  c.Get("User-Agent"),
		}

		// Add user info if available
		if userID := c.Locals("user_id"); userID != nil {
			logEntry["user_id"] = userID
		}
		if username := c.Locals("username"); username != nil {
			logEntry["username"] = username
		}

		// Add error if present
		if err != nil {
			logEntry["error"] = err.Error()
		}

		// Marshal to JSON
		logJSON, _ := json.Marshal(logEntry)
		logJSON = append(logJSON, '\n')

		// Write to stdout
		loggerWriter.Write(logJSON)

		return err
	}
}
