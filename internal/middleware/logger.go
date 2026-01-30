package middleware

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var loggerWriter io.Writer

func init() {
	// Initialize logger writer (file + console)
	logFile, err := os.OpenFile("logs/requests.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// If file can't be opened, use stdout
		loggerWriter = os.Stdout
		log.Printf("Warning: Failed to open request log file: %v. Logging to console only.", err)
	} else {
		// Write to both file and console
		loggerWriter = io.MultiWriter(os.Stdout, logFile)
	}
}

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

		// Write to log file and console
		if loggerWriter != nil {
			loggerWriter.Write(logJSON)
		} else {
			println(string(logJSON))
		}

		return err
	}
}
