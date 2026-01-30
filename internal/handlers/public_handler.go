package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rizkysr90/aslam-flower/internal/repositories"
	"github.com/rizkysr90/aslam-flower/internal/services"
)

// PublicHandler handles public catalog routes
type PublicHandler struct {
	productService  *services.ProductService
	categoryService *services.CategoryService
	whatsAppNumber  string
	storeName       string
	storeAddress    string
	shopeeLink      string
}

// NewPublicHandler creates a new public handler
func NewPublicHandler(productService *services.ProductService, categoryService *services.CategoryService, whatsAppNumber, storeName, storeAddress, shopeeLink string) *PublicHandler {
	return &PublicHandler{
		productService:  productService,
		categoryService: categoryService,
		whatsAppNumber:  whatsAppNumber,
		storeName:       storeName,
		storeAddress:    storeAddress,
		shopeeLink:      shopeeLink,
	}
}

// Landing renders the main catalog page with products and filters
func (h *PublicHandler) Landing(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get all categories for filter sidebar
	categories, err := h.categoryService.GetAll(ctx)
	if err != nil {
		return c.Status(500).SendString("Failed to load categories")
	}

	// Parse query parameters
	filters := h.parseFilters(c)

	// Get products with filters
	result, err := h.productService.GetAll(ctx, filters)
	if err != nil {
		return c.Status(500).SendString("Failed to load products")
	}

	// Render template
	return c.Render("pages/landing", fiber.Map{
		"Title":          "Katalog Produk",
		"ContentBlock":   "landing-content",
		"Products":       result.Products,
		"Categories":     categories,
		"Filters":        filters,
		"StoreName":      h.storeName,
		"StoreAddress":   h.storeAddress,
		"ShopeeLink":     h.shopeeLink,
		"WhatsAppNumber": h.whatsAppNumber,
		"Pagination": fiber.Map{
			"CurrentPage": result.Page,
			"TotalPages":  result.TotalPages,
			"Total":       result.Total,
			"PageSize":    result.PageSize,
		},
	}, "layouts/base")
}

// ProductDetail renders product detail page
func (h *PublicHandler) ProductDetail(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get product ID from URL params
	idParam := c.Params("id")
	productID, err := strconv.Atoi(idParam)
	if err != nil || productID <= 0 {
		return c.Status(404).SendString("Product not found")
	}

	// Load product with variants
	product, err := h.productService.GetByID(ctx, productID)
	if err != nil {
		return c.Status(404).SendString("Product not found")
	}

	// Render template
	return c.Render("pages/product-detail", fiber.Map{
		"Title":          product.Title,
		"ContentBlock":   "product-detail-content",
		"Product":        product,
		"WhatsAppNumber": h.whatsAppNumber,
	}, "layouts/base")
}

// SearchProducts handles product search (htmx partial)
func (h *PublicHandler) SearchProducts(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse search query from form
	query := c.FormValue("q")
	if query == "" {
		query = c.Query("q", "")
	}

	if query == "" {
		// Return empty grid if no query
		return c.Render("partials/product-grid", fiber.Map{
			"Products": []interface{}{},
		})
	}

	// Search products
	products, err := h.productService.Search(ctx, query)
	if err != nil {
		return c.Status(500).SendString("Search failed")
	}

	// Return HTML partial for htmx
	return c.Render("partials/product-grid", fiber.Map{
		"Products": products,
	})
}

// FilterProducts handles product filtering (htmx partial)
func (h *PublicHandler) FilterProducts(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse filter parameters (from form or query)
	filters := h.parseFilters(c)

	// Also parse from form values if POST request
	if c.Method() == "POST" {
		// Parse category from form
		if categoryStr := c.FormValue("category"); categoryStr != "" {
			if categoryID, err := strconv.Atoi(categoryStr); err == nil && categoryID > 0 {
				filters.CategoryID = &categoryID
			}
		}

		// Parse price range from form
		if minPriceStr := c.FormValue("price_min"); minPriceStr != "" {
			if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil && minPrice >= 0 {
				filters.MinPrice = &minPrice
			}
		}
		if maxPriceStr := c.FormValue("price_max"); maxPriceStr != "" {
			if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil && maxPrice >= 0 {
				filters.MaxPrice = &maxPrice
			}
		}

		// Parse availability from form
		// "available" = is_sold = false
		// "soldout" = is_sold = true
		availability := c.FormValue("availability")
		if availability == "available" {
			isSold := false
			filters.IsSold = &isSold
		} else if availability == "soldout" {
			isSold := true
			filters.IsSold = &isSold
		}

		// Parse sale from form
		if sale := c.FormValue("sale"); sale == "true" || sale == "1" {
			isSale := true
			filters.IsSale = &isSale
		}

		// Parse sort from form
		if sort := c.FormValue("sort"); sort != "" {
			validSorts := map[string]bool{
				"newest":     true,
				"price_asc":  true,
				"price_desc": true,
				"name_asc":   true,
			}
			if validSorts[sort] {
				filters.SortBy = sort
			}
		}
	}

	// Get products with filters
	result, err := h.productService.GetAll(ctx, filters)
	if err != nil {
		return c.Status(500).SendString("Failed to filter products")
	}

	// Return HTML partial for htmx
	return c.Render("partials/product-grid", fiber.Map{
		"Products": result.Products,
		"Pagination": fiber.Map{
			"CurrentPage": result.Page,
			"TotalPages":  result.TotalPages,
			"Total":       result.Total,
			"PageSize":    result.PageSize,
		},
	})
}

// parseFilters parses query parameters into ProductFilters
func (h *PublicHandler) parseFilters(c *fiber.Ctx) repositories.ProductFilters {
	filters := repositories.ProductFilters{
		PageSize: 20, // Default page size
		SortBy:   "newest",
	}

	// Parse page
	if pageStr := c.Query("page", "1"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filters.Page = page
		} else {
			filters.Page = 1
		}
	} else {
		filters.Page = 1
	}

	// Parse category filter
	if categoryStr := c.Query("category", ""); categoryStr != "" {
		if categoryID, err := strconv.Atoi(categoryStr); err == nil && categoryID > 0 {
			filters.CategoryID = &categoryID
		}
	}

	// Parse price range
	if minPriceStr := c.Query("price_min", ""); minPriceStr != "" {
		if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil && minPrice >= 0 {
			filters.MinPrice = &minPrice
		}
	}
	if maxPriceStr := c.Query("price_max", ""); maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil && maxPrice >= 0 {
			filters.MaxPrice = &maxPrice
		}
	}

	// Parse availability filter
	// "available" = is_sold = false
	// "soldout" = is_sold = true
	if availability := c.Query("availability", ""); availability != "" {
		switch availability {
		case "available":
			isSold := false
			filters.IsSold = &isSold
		case "soldout":
			isSold := true
			filters.IsSold = &isSold
		}
	}

	// Parse sale filter
	if sale := c.Query("sale", ""); sale == "true" || sale == "1" {
		isSale := true
		filters.IsSale = &isSale
	}

	// Parse sort
	if sort := c.Query("sort", ""); sort != "" {
		validSorts := map[string]bool{
			"newest":     true,
			"price_asc":  true,
			"price_desc": true,
			"name_asc":   true,
		}
		if validSorts[sort] {
			filters.SortBy = sort
		}
	}

	// Parse search query
	if searchQuery := c.Query("search", ""); searchQuery != "" {
		filters.SearchQuery = searchQuery
	}

	return filters
}
