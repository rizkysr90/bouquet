package handlers

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rizkysr90/aslam-flower/internal/models"
	"github.com/rizkysr90/aslam-flower/internal/repositories"
	"github.com/rizkysr90/aslam-flower/internal/services"
)

// getCSRFToken retrieves CSRF token from context or cookie
func getCSRFToken(c *fiber.Ctx) string {
	if token := c.Locals("csrf_token"); token != nil {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}
	return c.Cookies("csrf_")
}

// AdminHandler handles admin CRUD routes
type AdminHandler struct {
	productService    *services.ProductService
	categoryService   *services.CategoryService
	cloudinaryService *services.CloudinaryService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	productService *services.ProductService,
	categoryService *services.CategoryService,
	cloudinaryService *services.CloudinaryService,
) *AdminHandler {
	return &AdminHandler{
		productService:    productService,
		categoryService:   categoryService,
		cloudinaryService: cloudinaryService,
	}
}

// Dashboard renders admin dashboard with stats
func (h *AdminHandler) Dashboard(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get stats
	stats, err := h.getStats(ctx)
	if err != nil {
		return c.Status(500).SendString("Failed to load dashboard stats")
	}

	// Get recent products (last 5)
	recentFilters := repositories.ProductFilters{
		Page:     1,
		PageSize: 5,
		SortBy:   "newest",
	}
	recentResult, _ := h.productService.GetAll(ctx, recentFilters)

	return c.Render("pages/admin/dashboard", fiber.Map{
		"Title":          "Admin Dashboard",
		"Stats":          stats,
		"RecentProducts": recentResult.Products,
		"CSRFToken":      getCSRFToken(c),
		"CurrentPage":    "dashboard",
		"ContentBlock":   "admin-content-dashboard",
	}, "layouts/admin")
}

// ListProducts renders product list page
func (h *AdminHandler) ListProducts(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	page := 1
	if pageStr := c.Query("page", "1"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	searchQuery := c.Query("search", "")

	// Build filters
	filters := repositories.ProductFilters{
		Page:     page,
		PageSize: 50, // Admin uses 50 per page
		SortBy:   "newest",
	}

	if searchQuery != "" {
		filters.SearchQuery = searchQuery
	}

	// Get products
	result, err := h.productService.GetAll(ctx, filters)
	if err != nil {
		return c.Status(500).SendString("Failed to load products")
	}

	return c.Render("pages/admin/products", fiber.Map{
		"Title":    "Manage Products",
		"Products": result.Products,
		"Pagination": fiber.Map{
			"CurrentPage": result.Page,
			"TotalPages":  result.TotalPages,
			"Total":       result.Total,
			"PageSize":    result.PageSize,
		},
		"SearchQuery":  searchQuery,
		"CSRFToken":    getCSRFToken(c),
		"CurrentPage":  "products",
		"ContentBlock": "admin-content-products",
	}, "layouts/admin")
}

// NewProductForm renders product creation form
func (h *AdminHandler) NewProductForm(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get all categories for dropdown
	categories, err := h.categoryService.GetAll(ctx)
	if err != nil {
		return c.Status(500).SendString("Failed to load categories")
	}

	return c.Render("pages/admin/product-form", fiber.Map{
		"Title":        "Create Product",
		"Product":      nil,
		"Categories":   categories,
		"IsEdit":       false,
		"CSRFToken":    getCSRFToken(c),
		"CurrentPage":  "products",
		"ContentBlock": "admin-content-form",
	}, "layouts/admin")
}

// CloudinarySign returns signed parameters for direct browser upload to Cloudinary (JSON).
func (h *AdminHandler) CloudinarySign(c *fiber.Ctx) error {
	var body struct {
		Kind string `json:"kind"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
	}
	params, err := h.cloudinaryService.GenerateClientDirectUpload(strings.TrimSpace(body.Kind))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(params)
}

// CreateProduct handles product creation (images via client direct upload + hidden fields; no multipart files).
func (h *AdminHandler) CreateProduct(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse product data
	product := &models.Product{
		Code:        strings.TrimSpace(c.FormValue("code")),
		Title:       strings.TrimSpace(c.FormValue("title")),
		Description: strings.TrimSpace(c.FormValue("description")),
	}

	// Parse category ID
	if categoryStr := c.FormValue("category_id"); categoryStr != "" {
		if categoryID, err := strconv.Atoi(categoryStr); err == nil && categoryID > 0 {
			product.CategoryID = &categoryID
		}
	}

	// Parse base price
	if priceStr := c.FormValue("base_price"); priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			product.BasePrice = price
		}
	}

	// Parse is_sold flag
	product.IsSold = c.FormValue("is_sold") == "on" || c.FormValue("is_sold") == "true"

	mainURL := strings.TrimSpace(c.FormValue("main_photo_url"))
	mainPID := strings.TrimSpace(c.FormValue("main_photo_id"))
	if mainURL != "" || mainPID != "" {
		if mainURL == "" || mainPID == "" {
			return c.Status(400).SendString("Main photo: provide both URL and public ID, or leave both empty")
		}
		if err := h.cloudinaryService.ValidateClientUploadResult("main", mainURL, mainPID); err != nil {
			return c.Status(400).SendString("Invalid main photo: " + err.Error())
		}
		product.MainPhotoURL = mainURL
		product.MainPhotoID = mainPID
	}

	// Parse variants from form: only variants[N][field] — parallel variants[][photo] is unsafe
	// because empty file inputs are omitted and indices no longer match row order.
	var variants []models.ProductVariant

	// Handle indexed format: variants[N][color]
	indexedVariantsMap := make(map[int]map[string]string)
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		values := []string{string(value)}
		if strings.HasPrefix(keyStr, "variants[") && strings.Contains(keyStr, "][") && !strings.Contains(keyStr, "variants[][") {
			start := strings.Index(keyStr, "[") + 1
			middle := strings.Index(keyStr, "][")
			if start > 0 && middle > start {
				indexStr := keyStr[start:middle]
				if index, err := strconv.Atoi(indexStr); err == nil {
					fieldEnd := strings.Index(keyStr[middle+2:], "]")
					if fieldEnd > 0 {
						field := keyStr[middle+2 : middle+2+fieldEnd]
						if len(values) > 0 {
							if indexedVariantsMap[index] == nil {
								indexedVariantsMap[index] = make(map[string]string)
							}
							if field == "color" {
								color := strings.TrimSpace(values[0])
								if color != "" {
									indexedVariantsMap[index][field] = color
								}
							} else if field == "price_adjustment" && values[0] != "" {
								indexedVariantsMap[index][field] = strings.TrimSpace(values[0])
							} else if field == "is_sale" {
								if values[0] == "on" || values[0] == "true" {
									indexedVariantsMap[index][field] = "true"
								} else {
									indexedVariantsMap[index][field] = "false"
								}
							} else if field == "photo_url" && len(values) > 0 {
								indexedVariantsMap[index][field] = strings.TrimSpace(values[0])
							} else if field == "photo_id" && len(values) > 0 {
								indexedVariantsMap[index][field] = strings.TrimSpace(values[0])
							}
						}
					}
				}
			}
		}
	})

	// Add indexed variants
	indexedIndices := make([]int, 0, len(indexedVariantsMap))
	for idx := range indexedVariantsMap {
		indexedIndices = append(indexedIndices, idx)
	}
	sort.Ints(indexedIndices)

	for _, index := range indexedIndices {
		variantData := indexedVariantsMap[index]
		color := variantData["color"]
		if color == "" {
			continue
		}
		variant := models.ProductVariant{Color: color}
		// Admin form input is treated as FINAL variant price (what should be shown publicly).
		// Store it as price_adjustment so public can compute FinalPrice = base + adjustment.
		if finalPriceStr, ok := variantData["price_adjustment"]; ok && finalPriceStr != "" {
			if finalPrice, err := strconv.ParseFloat(finalPriceStr, 64); err == nil {
				variant.PriceAdjustment = finalPrice - product.BasePrice
			}
		}
		if isSaleStr, ok := variantData["is_sale"]; ok {
			variant.IsSale = isSaleStr == "true"
		}
		pu := variantData["photo_url"]
		pid := variantData["photo_id"]
		if pu != "" || pid != "" {
			if pu == "" || pid == "" {
				return c.Status(400).SendString(fmt.Sprintf("Variant %q: provide both photo URL and public ID, or leave both empty", color))
			}
			if err := h.cloudinaryService.ValidateClientUploadResult("variant", pu, pid); err != nil {
				return c.Status(400).SendString("Invalid variant image for " + color + ": " + err.Error())
			}
			variant.PhotoURL = pu
			variant.PhotoID = pid
		}
		variants = append(variants, variant)
	}

	product.Variants = variants

	if err := h.productService.Create(ctx, product, nil, ""); err != nil {
		log.Printf("ERROR: Failed to create product: %v", err)
		return c.Status(400).SendString(fmt.Sprintf("Failed to create product: %v", err))
	}

	return c.Redirect("/admin/products")
}

// EditProductForm renders product edit form
func (h *AdminHandler) EditProductForm(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get product ID
	idParam := c.Params("id")
	productID, err := strconv.Atoi(idParam)
	if err != nil || productID <= 0 {
		return c.Status(404).SendString("Product not found")
	}

	// Get product
	product, err := h.productService.GetByID(ctx, productID)
	if err != nil {
		return c.Status(404).SendString("Product not found")
	}

	// Get all categories
	categories, err := h.categoryService.GetAll(ctx)
	if err != nil {
		return c.Status(500).SendString("Failed to load categories")
	}

	return c.Render("pages/admin/product-form", fiber.Map{
		"Title":        "Edit Product",
		"Product":      product,
		"Categories":   categories,
		"IsEdit":       true,
		"CSRFToken":    getCSRFToken(c),
		"CurrentPage":  "products",
		"ContentBlock": "admin-content-form",
	}, "layouts/admin")
}

// UpdateProduct handles product update
func (h *AdminHandler) UpdateProduct(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get product ID
	idParam := c.Params("id")
	productID, err := strconv.Atoi(idParam)
	if err != nil || productID <= 0 {
		return c.Status(404).SendString("Product not found")
	}

	// Get existing product to preserve variant photos
	existingProduct, err := h.productService.GetByID(ctx, productID)
	if err != nil {
		return c.Status(404).SendString("Product not found")
	}

	// Create a map of existing variants by color for photo preservation
	existingVariantsMap := make(map[string]models.ProductVariant)
	for _, v := range existingProduct.Variants {
		existingVariantsMap[v.Color] = v
	}

	// Parse product data
	product := &models.Product{
		ID:          productID,
		Code:        strings.TrimSpace(c.FormValue("code")),
		Title:       strings.TrimSpace(c.FormValue("title")),
		Description: strings.TrimSpace(c.FormValue("description")),
	}

	// Parse category ID
	if categoryStr := c.FormValue("category_id"); categoryStr != "" {
		if categoryID, err := strconv.Atoi(categoryStr); err == nil && categoryID > 0 {
			product.CategoryID = &categoryID
		} else {
			product.CategoryID = nil
		}
	} else {
		product.CategoryID = nil
	}

	// Parse base price
	if priceStr := c.FormValue("base_price"); priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			product.BasePrice = price
		}
	}

	mainURL := strings.TrimSpace(c.FormValue("main_photo_url"))
	mainPID := strings.TrimSpace(c.FormValue("main_photo_id"))
	if mainURL != "" || mainPID != "" {
		if mainURL == "" || mainPID == "" {
			return c.Status(400).SendString("Main photo: provide both URL and public ID, or leave both empty")
		}
		if err := h.cloudinaryService.ValidateClientUploadResult("main", mainURL, mainPID); err != nil {
			return c.Status(400).SendString("Invalid main photo: " + err.Error())
		}
		product.MainPhotoURL = mainURL
		product.MainPhotoID = mainPID
	}

	// Parse variants from form: only variants[N][field] (see CreateProduct comment).
	var variants []models.ProductVariant

	// Collect variant fields: variants[N][color], etc.
	indexedVariantsMap := make(map[int]map[string]string)
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		values := []string{string(value)}
		if strings.HasPrefix(keyStr, "variants[") && strings.Contains(keyStr, "][") && !strings.Contains(keyStr, "variants[][") {
			start := strings.Index(keyStr, "[") + 1
			middle := strings.Index(keyStr, "][")
			if start > 0 && middle > start {
				indexStr := keyStr[start:middle]
				if index, err := strconv.Atoi(indexStr); err == nil {
					fieldEnd := strings.Index(keyStr[middle+2:], "]")
					if fieldEnd > 0 {
						field := keyStr[middle+2 : middle+2+fieldEnd]
						if len(values) > 0 {
							if indexedVariantsMap[index] == nil {
								indexedVariantsMap[index] = make(map[string]string)
							}
							if field == "color" {
								color := strings.TrimSpace(values[0])
								if color != "" {
									indexedVariantsMap[index][field] = color
								}
							} else if field == "price_adjustment" && values[0] != "" {
								indexedVariantsMap[index][field] = strings.TrimSpace(values[0])
							} else if field == "is_sale" {
								if values[0] == "on" || values[0] == "true" {
									indexedVariantsMap[index][field] = "true"
								} else {
									indexedVariantsMap[index][field] = "false"
								}
							} else if field == "photo_url" && len(values) > 0 {
								indexedVariantsMap[index][field] = strings.TrimSpace(values[0])
							} else if field == "photo_id" && len(values) > 0 {
								indexedVariantsMap[index][field] = strings.TrimSpace(values[0])
							}
						}
					}
				}
			}
		}
	})

	// Convert indexed map to sorted slice
	indexedIndices := make([]int, 0, len(indexedVariantsMap))
	for idx := range indexedVariantsMap {
		indexedIndices = append(indexedIndices, idx)
	}
	sort.Ints(indexedIndices)

	// Build variants from indexed rows only (edit + JS-added rows)
	for _, index := range indexedIndices {
		variantData := indexedVariantsMap[index]
		color := variantData["color"]
		if color == "" {
			continue
		}

		variant := models.ProductVariant{
			Color: color,
		}

		// Admin form input is treated as FINAL variant price (what should be shown publicly).
		// Store it as price_adjustment so public can compute FinalPrice = base + adjustment.
		if finalPriceStr, ok := variantData["price_adjustment"]; ok && finalPriceStr != "" {
			if finalPrice, err := strconv.ParseFloat(finalPriceStr, 64); err == nil {
				variant.PriceAdjustment = finalPrice - product.BasePrice
			}
		}

		// Parse is_sale flag
		if isSaleStr, ok := variantData["is_sale"]; ok {
			variant.IsSale = isSaleStr == "true"
		}

		pu := variantData["photo_url"]
		pid := variantData["photo_id"]
		if pu != "" || pid != "" {
			if pu == "" || pid == "" {
				return c.Status(400).SendString(fmt.Sprintf("Variant %q: provide both photo URL and public ID, or leave both empty", color))
			}
			if err := h.cloudinaryService.ValidateClientUploadResult("variant", pu, pid); err != nil {
				return c.Status(400).SendString("Invalid variant image for " + color + ": " + err.Error())
			}
			variant.PhotoURL = pu
			variant.PhotoID = pid
		} else if existingVariant, exists := existingVariantsMap[color]; exists {
			variant.PhotoURL = existingVariant.PhotoURL
			variant.PhotoID = existingVariant.PhotoID
		}

		variants = append(variants, variant)
	}

	product.Variants = variants

	err = h.productService.Update(ctx, productID, product, nil, "")
	if err != nil {
		return c.Status(400).SendString(fmt.Sprintf("Failed to update product: %v", err))
	}

	return c.Redirect("/admin/products")
}

// DeleteProduct handles product deletion (htmx)
func (h *AdminHandler) DeleteProduct(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get product ID
	idParam := c.Params("id")
	productID, err := strconv.Atoi(idParam)
	if err != nil || productID <= 0 {
		return c.Status(400).SendString("Invalid product ID")
	}

	// Delete product
	err = h.productService.Delete(ctx, productID)
	if err != nil {
		return c.Status(500).SendString(fmt.Sprintf("Failed to delete product: %v", err))
	}

	// Return success response for htmx
	return c.SendString("Product deleted successfully")
}

// getStats retrieves dashboard statistics
func (h *AdminHandler) getStats(ctx context.Context) (map[string]interface{}, error) {
	// Get total products
	productsResult, err := h.productService.GetAll(ctx, repositories.ProductFilters{
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		return nil, err
	}

	// Get total categories
	categories, err := h.categoryService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"TotalProducts":   productsResult.Total,
		"TotalCategories": len(categories),
	}, nil
}
