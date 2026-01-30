package handlers

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"sort"
	"strconv"
	"strings"
	"time"

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

// CreateProduct handles product creation
func (h *AdminHandler) CreateProduct(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse multipart form (max 10MB for file upload)
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).SendString("Invalid form data")
	}

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

	// Get main photo file
	var mainPhotoFile multipart.File
	var photoFilename string
	if files := form.File["main_photo"]; len(files) > 0 && files[0].Size > 0 {
		fileHeader := files[0]
		log.Printf("Uploading file: name=%s, size=%d, type=%s", fileHeader.Filename, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("ERROR: Failed to open photo file: %v", err)
			return c.Status(400).SendString(fmt.Sprintf("Failed to open photo file: %v", err))
		}
		mainPhotoFile = file
		photoFilename = fileHeader.Filename

		// Generate unique filename using timestamp (without folder, Cloudinary service adds it)
		photoFilename = fmt.Sprintf("%s-%d", product.Code, time.Now().Unix())
		log.Printf("Generated photo filename: %s", photoFilename)
	} else {
		// Log if no file was uploaded (for debugging)
		if len(form.File["main_photo"]) == 0 {
			log.Printf("WARNING: No file field 'main_photo' in form")
		} else if len(form.File["main_photo"]) > 0 && form.File["main_photo"][0].Size == 0 {
			log.Printf("WARNING: File field 'main_photo' exists but is empty (size=0)")
		}
	}

	// Parse variants from form (same logic as UpdateProduct)
	var variants []models.ProductVariant

	// Handle non-indexed format: variants[][color]
	if colors := form.Value["variants[][color]"]; len(colors) > 0 {
		priceAdjustments := form.Value["variants[][price_adjustment]"]
		isSales := form.Value["variants[][is_sale]"]
		nonIndexedPhotos := form.File["variants[][photo]"]
		for i, color := range colors {
			color = strings.TrimSpace(color)
			if color == "" {
				continue
			}
			variant := models.ProductVariant{Color: color}
			if i < len(priceAdjustments) && priceAdjustments[i] != "" {
				if priceAdj, err := strconv.ParseFloat(strings.TrimSpace(priceAdjustments[i]), 64); err == nil {
					variant.PriceAdjustment = priceAdj
				}
			}
			// Parse is_sale flag
			if i < len(isSales) && (isSales[i] == "on" || isSales[i] == "true") {
				variant.IsSale = true
			}
			// Handle photo
			if i < len(nonIndexedPhotos) && nonIndexedPhotos[i].Size > 0 {
				photo, err := nonIndexedPhotos[i].Open()
				if err == nil {
					photoFilename := fmt.Sprintf("%s-%s-%d", product.Code, color, time.Now().Unix())
					photoURL, photoID, err := h.cloudinaryService.UploadProductImage(ctx, photo, photoFilename)
					photo.Close()
					if err == nil {
						variant.PhotoURL = photoURL
						variant.PhotoID = photoID
					}
				}
			}
			variants = append(variants, variant)
		}
	}

	// Handle indexed format: variants[N][color]
	indexedVariantsMap := make(map[int]map[string]string)
	for key, values := range form.Value {
		if strings.HasPrefix(key, "variants[") && strings.Contains(key, "][") && !strings.Contains(key, "variants[][") {
			start := strings.Index(key, "[") + 1
			middle := strings.Index(key, "][")
			if start > 0 && middle > start {
				indexStr := key[start:middle]
				if index, err := strconv.Atoi(indexStr); err == nil {
					fieldEnd := strings.Index(key[middle+2:], "]")
					if fieldEnd > 0 {
						field := key[middle+2 : middle+2+fieldEnd]
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
							}
						}
					}
				}
			}
		}
	}

	// Process indexed photos
	indexedPhotosMap := make(map[int]*multipart.FileHeader)
	for key, files := range form.File {
		if strings.HasPrefix(key, "variants[") && strings.Contains(key, "][photo]") && !strings.Contains(key, "variants[][") {
			start := strings.Index(key, "[") + 1
			middle := strings.Index(key, "][photo]")
			if start > 0 && middle > start {
				indexStr := key[start:middle]
				if index, err := strconv.Atoi(indexStr); err == nil {
					if len(files) > 0 && files[0].Size > 0 {
						indexedPhotosMap[index] = files[0]
					}
				}
			}
		}
	}

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
		if priceAdjStr, ok := variantData["price_adjustment"]; ok && priceAdjStr != "" {
			if priceAdj, err := strconv.ParseFloat(priceAdjStr, 64); err == nil {
				variant.PriceAdjustment = priceAdj
			}
		}
		if isSaleStr, ok := variantData["is_sale"]; ok {
			variant.IsSale = isSaleStr == "true"
		}
		if photoFile, ok := indexedPhotosMap[index]; ok {
			photo, err := photoFile.Open()
			if err == nil {
				photoFilename := fmt.Sprintf("%s-%s-%d", product.Code, color, time.Now().Unix())
				photoURL, photoID, err := h.cloudinaryService.UploadProductImage(ctx, photo, photoFilename)
				photo.Close()
				if err == nil {
					variant.PhotoURL = photoURL
					variant.PhotoID = photoID
				}
			}
		}
		variants = append(variants, variant)
	}

	product.Variants = variants

	// Create product (service will handle file closing)
	err = h.productService.Create(ctx, product, mainPhotoFile, photoFilename)
	if mainPhotoFile != nil {
		mainPhotoFile.Close() // Close after service is done
	}
	if err != nil {
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

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).SendString("Invalid form data")
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

	// Check if new photo is uploaded
	var newPhotoFile multipart.File
	var photoFilename string
	if files := form.File["main_photo"]; len(files) > 0 && files[0].Size > 0 {
		fileHeader := files[0]
		file, err := fileHeader.Open()
		if err != nil {
			return c.Status(400).SendString("Failed to process photo")
		}
		newPhotoFile = file
		photoFilename = fileHeader.Filename

		// Generate unique filename using timestamp
		photoFilename = fmt.Sprintf("%s-%d", product.Code, time.Now().Unix())
	}

	// Parse variants from form
	// Form uses both variants[][color] (existing) and variants[N][color] (new via JS)
	var variants []models.ProductVariant

	// Step 1: Collect all variant data by parsing form values
	// Handle non-indexed format: variants[][color]
	nonIndexedVariants := []map[string]string{}
	if colors := form.Value["variants[][color]"]; len(colors) > 0 {
		priceAdjustments := form.Value["variants[][price_adjustment]"]
		isSales := form.Value["variants[][is_sale]"]
		for i, color := range colors {
			color = strings.TrimSpace(color)
			if color == "" {
				continue
			}
			variantData := map[string]string{"color": color}
			if i < len(priceAdjustments) && priceAdjustments[i] != "" {
				variantData["price_adjustment"] = strings.TrimSpace(priceAdjustments[i])
			}
			// Parse is_sale flag (checkbox: "on" or "true" = true, otherwise false)
			if i < len(isSales) && (isSales[i] == "on" || isSales[i] == "true") {
				variantData["is_sale"] = "true"
			} else {
				variantData["is_sale"] = "false"
			}
			nonIndexedVariants = append(nonIndexedVariants, variantData)
		}
	}

	// Handle indexed format: variants[N][color]
	indexedVariantsMap := make(map[int]map[string]string)
	for key, values := range form.Value {
		// Check for indexed format: variants[N][color] or variants[N][price_adjustment]
		if strings.HasPrefix(key, "variants[") && strings.Contains(key, "][") && !strings.Contains(key, "variants[][") {
			// Extract index: variants[N][field]
			start := strings.Index(key, "[") + 1
			middle := strings.Index(key, "][")
			if start > 0 && middle > start {
				indexStr := key[start:middle]
				if index, err := strconv.Atoi(indexStr); err == nil {
					// Extract field name
					fieldEnd := strings.Index(key[middle+2:], "]")
					if fieldEnd > 0 {
						field := key[middle+2 : middle+2+fieldEnd]
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
								// Parse is_sale flag (checkbox: "on" or "true" = true, otherwise false)
								if values[0] == "on" || values[0] == "true" {
									indexedVariantsMap[index][field] = "true"
								} else {
									indexedVariantsMap[index][field] = "false"
								}
							}
						}
					}
				}
			}
		}
	}

	// Convert indexed map to sorted slice
	indexedIndices := make([]int, 0, len(indexedVariantsMap))
	for idx := range indexedVariantsMap {
		indexedIndices = append(indexedIndices, idx)
	}
	sort.Ints(indexedIndices)

	// Step 2: Process variant photos
	// Non-indexed photos
	nonIndexedPhotos := form.File["variants[][photo]"]

	// Indexed photos
	indexedPhotosMap := make(map[int]*multipart.FileHeader)
	for key, files := range form.File {
		if strings.HasPrefix(key, "variants[") && strings.Contains(key, "][photo]") && !strings.Contains(key, "variants[][") {
			start := strings.Index(key, "[") + 1
			middle := strings.Index(key, "][photo]")
			if start > 0 && middle > start {
				indexStr := key[start:middle]
				if index, err := strconv.Atoi(indexStr); err == nil {
					if len(files) > 0 && files[0].Size > 0 {
						indexedPhotosMap[index] = files[0]
					}
				}
			}
		}
	}

	// Step 3: Build variants array
	// First add non-indexed variants (existing)
	for i, variantData := range nonIndexedVariants {
		color := variantData["color"]
		if color == "" {
			continue
		}

		variant := models.ProductVariant{
			Color: color,
		}

		if priceAdjStr, ok := variantData["price_adjustment"]; ok && priceAdjStr != "" {
			if priceAdj, err := strconv.ParseFloat(priceAdjStr, 64); err == nil {
				variant.PriceAdjustment = priceAdj
			}
		}

		// Parse is_sale flag
		if isSaleStr, ok := variantData["is_sale"]; ok {
			variant.IsSale = isSaleStr == "true"
		}

		// Handle photo for non-indexed variant
		if i < len(nonIndexedPhotos) && nonIndexedPhotos[i].Size > 0 {
			// New photo uploaded - upload it
			photo, err := nonIndexedPhotos[i].Open()
			if err == nil {
				photoFilename := fmt.Sprintf("%s-%s-%d", product.Code, color, time.Now().Unix())
				photoURL, photoID, err := h.cloudinaryService.UploadProductImage(ctx, photo, photoFilename)
				photo.Close()
				if err == nil {
					variant.PhotoURL = photoURL
					variant.PhotoID = photoID
				}
			}
		} else {
			// No new photo uploaded - preserve existing photo if variant exists
			if existingVariant, exists := existingVariantsMap[color]; exists {
				variant.PhotoURL = existingVariant.PhotoURL
				variant.PhotoID = existingVariant.PhotoID
			}
		}

		variants = append(variants, variant)
	}

	// Then add indexed variants (new)
	for _, index := range indexedIndices {
		variantData := indexedVariantsMap[index]
		color := variantData["color"]
		if color == "" {
			continue
		}

		variant := models.ProductVariant{
			Color: color,
		}

		if priceAdjStr, ok := variantData["price_adjustment"]; ok && priceAdjStr != "" {
			if priceAdj, err := strconv.ParseFloat(priceAdjStr, 64); err == nil {
				variant.PriceAdjustment = priceAdj
			}
		}

		// Parse is_sale flag
		if isSaleStr, ok := variantData["is_sale"]; ok {
			variant.IsSale = isSaleStr == "true"
		}

		// Handle photo for indexed variant
		if photoFile, ok := indexedPhotosMap[index]; ok {
			photo, err := photoFile.Open()
			if err == nil {
				photoFilename := fmt.Sprintf("%s-%s-%d", product.Code, color, time.Now().Unix())
				photoURL, photoID, err := h.cloudinaryService.UploadProductImage(ctx, photo, photoFilename)
				photo.Close()
				if err == nil {
					variant.PhotoURL = photoURL
					variant.PhotoID = photoID
				}
			}
		}

		variants = append(variants, variant)
	}

	product.Variants = variants

	// Update product (service will handle file closing)
	err = h.productService.Update(ctx, productID, product, newPhotoFile, photoFilename)
	if newPhotoFile != nil {
		newPhotoFile.Close() // Close after service is done
	}
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
