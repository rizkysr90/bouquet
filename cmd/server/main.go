package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/rizkysr90/aslam-flower/internal/config"
	"github.com/rizkysr90/aslam-flower/internal/handlers"
	"github.com/rizkysr90/aslam-flower/internal/middleware"
	"github.com/rizkysr90/aslam-flower/internal/repositories"
	"github.com/rizkysr90/aslam-flower/internal/services"
)

func main() {
	// Initialize logging to stdout only
	initLogging()

	// Load configuration
	cfg := config.Load()

	// Validate required environment variables
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize database connection with connection pooling
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Printf("Database connection established (pool: max_open=%d max_idle=%d max_lifetime=%s max_idle_time=%s)",
		cfg.DBMaxOpenConns, cfg.DBMaxIdleConns, cfg.DBConnMaxLifetime, cfg.DBConnMaxIdleTime)

	// Migrations are run by dbmate (see docker-compose db-migration service or: dbmate up)

	// Initialize Cloudinary service
	cloudinaryService, err := services.NewCloudinaryService(cfg.CloudName, cfg.APIKey, cfg.APISecret)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudinary: %v", err)
	}
	apiKeyPreview := cfg.APIKey
	if len(apiKeyPreview) > 4 {
		apiKeyPreview = apiKeyPreview[:4]
	}
	log.Printf("Cloudinary service initialized - CloudName: %s, APIKey: %s***", cfg.CloudName, apiKeyPreview)

	// Initialize repositories
	productRepo := repositories.NewProductRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	adminRepo := repositories.NewAdminRepository(db)

	// Initialize services
	productService := services.NewProductService(productRepo, cloudinaryService, db)
	categoryService := services.NewCategoryService(categoryRepo)
	authService := services.NewAuthService(adminRepo, cfg.JWTSecret)

	// Initialize handlers
	publicHandler := handlers.NewPublicHandler(productService, categoryService, cfg.WhatsAppNumber, cfg.StoreName, cfg.StoreAddress, cfg.ShopeeLink)
	adminHandler := handlers.NewAdminHandler(productService, categoryService, cloudinaryService)
	categoryHandler := handlers.NewCategoryHandler(categoryService, categoryRepo)
	authHandler := handlers.NewAuthHandler(authService)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
		Views:        initTemplateEngine(cfg.Env),
		BodyLimit:    10 * 1024 * 1024, // 10MB for file uploads
	})

	// Register global middleware
	app.Use(recover.New())                // Panic recovery
	app.Use(middleware.Logger())          // Request logging
	app.Use(middleware.SecurityHeaders()) // Security headers
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // Allow all for public catalog
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-CSRF-Token",
	}))

	// CSRF protection (exclude public routes)
	// KeyLookup supports: "header:<name>", "form:<name>", "query:<name>", "param:<name>", "cookie:<name>"
	// Using custom extractor to support both form (for regular forms) and header (for htmx DELETE requests)
	csrfMiddleware := csrf.New(csrf.Config{
		KeyLookup:      "form:_csrf", // Default to form for regular forms
		CookieName:     "csrf_",
		CookieSameSite: "Strict",
		Expiration:     1 * time.Hour,
		ContextKey:     "csrf_token", // Store token in context for easy access
		// Custom extractor that checks both form and header
		Extractor: func(c *fiber.Ctx) (string, error) {
			// First try form field (for regular POST forms)
			if token := c.FormValue("_csrf"); token != "" {
				return token, nil
			}
			// Then try header (for htmx DELETE requests)
			if token := c.Get("X-CSRF-Token"); token != "" {
				return token, nil
			}
			// If neither found, return error
			return "", fiber.ErrForbidden
		},
	})

	// Health check endpoint (no middleware)
	app.Get("/health", func(c *fiber.Ctx) error {
		// Test database connection
		if err := db.Ping(); err != nil {
			return c.Status(503).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  "database connection failed",
			})
		}
		return c.JSON(fiber.Map{
			"status": "healthy",
		})
	})

	// Static files with correct MIME types (fasthttp serves .css/.js as text/plain)
	app.Get("/static/*", staticFileHandler("./web/static"))

	// Public routes (no CSRF, no auth)
	app.Get("/", publicHandler.Landing)
	app.Get("/products/:id", publicHandler.ProductDetail)
	app.Post("/products/search", publicHandler.SearchProducts)
	app.Post("/products/filter", publicHandler.FilterProducts)

	// Admin login routes (CSRF needed on GET to generate token, and on POST to validate)
	app.Get("/admin/login", csrfMiddleware, authHandler.LoginPage)
	app.Post("/admin/login", csrfMiddleware, authHandler.Login)
	app.Post("/admin/logout", authHandler.Logout)

	// Protected admin routes (CSRF + Auth required)
	adminGroup := app.Group("/admin", csrfMiddleware, middleware.AuthRequired(authService))
	adminGroup.Get("/dashboard", adminHandler.Dashboard)
	adminGroup.Get("/products", adminHandler.ListProducts)
	adminGroup.Get("/products/new", adminHandler.NewProductForm)
	adminGroup.Post("/products", adminHandler.CreateProduct)
	adminGroup.Get("/products/:id/edit", adminHandler.EditProductForm)
	adminGroup.Post("/products/:id", adminHandler.UpdateProduct)
	adminGroup.Post("/products/:id/delete", adminHandler.DeleteProduct)

	// Admin category routes
	adminGroup.Get("/categories", categoryHandler.ListCategories)
	adminGroup.Get("/categories/new", categoryHandler.NewCategoryForm)
	adminGroup.Post("/categories", categoryHandler.CreateCategory)
	adminGroup.Get("/categories/:id/edit", categoryHandler.EditCategoryForm)
	adminGroup.Post("/categories/:id", categoryHandler.UpdateCategory)
	adminGroup.Delete("/categories/:id", categoryHandler.DeleteCategory)

	// Start server with graceful shutdown
	startServer(app, cfg.Port)
}

// initDatabase initializes PostgreSQL connection with a configured connection pool.
// Pool limits connection count and recycles connections to avoid exhausting DB resources.
func initDatabase(cfg *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	// Connection pool (configurable via DB_MAX_OPEN_CONNS, DB_MAX_IDLE_CONNS, etc.)
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// initTemplateEngine initializes HTML template engine
func initTemplateEngine(env string) *html.Engine {
	engine := html.New("./web/templates", ".html")

	// Enable template reload in development
	if env == "development" {
		engine.Reload(true)
	}

	// Add custom template functions
	engine.AddFunc("formatPrice", func(price float64) string {
		// Format price with thousand separators
		priceInt := int64(price)
		return fmt.Sprintf("Rp %s", formatNumber(priceInt))
	})

	// Math functions for pagination
	engine.AddFunc("add", func(a, b int) int {
		return a + b
	})
	engine.AddFunc("sub", func(a, b int) int {
		return a - b
	})
	engine.AddFunc("seq", func(start, end int) []int {
		if start > end {
			return []int{}
		}
		result := make([]int, end-start+1)
		for i := range result {
			result[i] = start + i
		}
		return result
	})
	engine.AddFunc("deref", func(b *bool) bool {
		if b == nil {
			return false
		}
		return *b
	})
	engine.AddFunc("derefInt", func(i *int) int {
		if i == nil {
			return 0
		}
		return *i
	})

	return engine
}

// customErrorHandler handles application errors
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log detailed error
	log.Printf("ERROR [%s] %s: %v", c.Method(), c.Path(), err)

	// Return error response
	if c.Get("Accept") == "application/json" {
		return c.Status(code).JSON(fiber.Map{
			"error": message,
		})
	}

	return c.Status(code).SendString(message)
}

// staticFileHandler serves files from root with correct MIME types (fasthttp serves .css/.js as text/plain).
func staticFileHandler(root string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Params("*")
		if path == "" {
			return c.Status(404).SendString("Not found")
		}
		cleanPath := filepath.Clean(filepath.FromSlash(path))
		if strings.Contains(cleanPath, "..") {
			return c.Status(404).SendString("Not found")
		}
		fullPath := filepath.Join(root, cleanPath)
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			return c.Status(404).SendString("Not found")
		}
		if err := c.SendFile(fullPath); err != nil {
			return err
		}
		ext := filepath.Ext(fullPath)
		if ext != "" {
			switch ext {
			case ".css":
				c.Set("Content-Type", "text/css; charset=utf-8")
			case ".js":
				c.Set("Content-Type", "application/javascript; charset=utf-8")
			default:
				c.Type(ext)
			}
		}
		return nil
	}
}

// startServer starts the server with graceful shutdown
func startServer(app *fiber.App, port string) {
	// Create channel for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

// initLogging configures logging to stdout only
func initLogging() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Logging initialized - stdout only")
}

// formatNumber formats a number with thousand separators
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%d", n) // Simple version, can be enhanced with proper formatting
}
