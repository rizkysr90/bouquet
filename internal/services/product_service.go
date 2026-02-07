package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"

	"github.com/jmoiron/sqlx"
	"github.com/rizkysr90/aslam-flower/internal/models"
	"github.com/rizkysr90/aslam-flower/internal/repositories"
)

// ProductService handles product business logic
type ProductService struct {
	productRepo       *repositories.ProductRepository
	cloudinaryService *CloudinaryService
	db                *sqlx.DB
}

// NewProductService creates a new product service
func NewProductService(productRepo *repositories.ProductRepository, cloudinaryService *CloudinaryService, db *sqlx.DB) *ProductService {
	return &ProductService{
		productRepo:       productRepo,
		cloudinaryService: cloudinaryService,
		db:                db,
	}
}

// GetAll retrieves products with filtering, sorting, and pagination
func (s *ProductService) GetAll(ctx context.Context, filters repositories.ProductFilters) (*repositories.ProductListResult, error) {
	// Validate pagination
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}
	if filters.PageSize > 100 {
		filters.PageSize = 100 // Max page size
	}

	// Validate sort option
	validSorts := map[string]bool{
		"newest":     true,
		"price_asc":  true,
		"price_desc": true,
		"name_asc":   true,
	}
	if filters.SortBy != "" && !validSorts[filters.SortBy] {
		filters.SortBy = "newest" // Default to newest
	}

	result, err := s.productRepo.FindAll(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}

	return result, nil
}

// GetByID retrieves a product by ID with variants
func (s *ProductService) GetByID(ctx context.Context, id int) (*models.Product, error) {
	if id <= 0 {
		return nil, errors.New("invalid product ID")
	}

	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product: %w", err)
	}

	return product, nil
}

// Create creates a new product with photo upload
func (s *ProductService) Create(ctx context.Context, product *models.Product, mainPhoto multipart.File, photoFilename string) error {
	// Validate product data
	if err := s.validateProduct(product); err != nil {
		return err
	}

	// Check if product code already exists
	existing, _ := s.productRepo.FindByCode(product.Code)
	if existing != nil {
		return fmt.Errorf("product with code %s already exists", product.Code)
	}

	// Start transaction
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Upload photo to Cloudinary first
	var photoURL, photoID string
	if mainPhoto != nil {
		photoURL, photoID, err = s.cloudinaryService.UploadProductImage(ctx, mainPhoto, photoFilename)
		if err != nil {
			return fmt.Errorf("failed to upload photo to Cloudinary: %w", err)
		}
		if photoURL == "" || photoID == "" {
			return fmt.Errorf("Cloudinary upload returned empty URL or ID")
		}
		product.MainPhotoURL = photoURL
		product.MainPhotoID = photoID
	}

	// Create product in database
	// Note: We need to use the transaction, but repository uses DB directly
	// For now, we'll create the product and handle rollback manually if variants fail
	err = s.productRepo.Create(product)
	if err != nil {
		// Rollback: delete uploaded photo if product creation fails
		if photoID != "" {
			_ = s.cloudinaryService.DeleteImage(ctx, photoID)
		}
		return fmt.Errorf("failed to create product: %w", err)
	}

	// Create variants if provided
	if len(product.Variants) > 0 {
		// Validate variants before creating
		for _, variant := range product.Variants {
			if variant.Color == "" {
				// Rollback: delete product photo and product
				if photoID != "" {
					_ = s.cloudinaryService.DeleteImage(ctx, photoID)
				}
				_ = s.productRepo.Delete(product.ID)
				return errors.New("variant color is required")
			}
		}

		err = s.productRepo.CreateVariants(tx, product.ID, product.Variants)
		if err != nil {
			// Rollback: delete product photo and product
			if photoID != "" {
				_ = s.cloudinaryService.DeleteImage(ctx, photoID)
			}
			_ = s.productRepo.Delete(product.ID)
			return fmt.Errorf("failed to create variants: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		// Rollback: delete uploaded photo
		if photoID != "" {
			_ = s.cloudinaryService.DeleteImage(ctx, photoID)
		}
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Update updates an existing product
func (s *ProductService) Update(ctx context.Context, id int, product *models.Product, newPhoto multipart.File, photoFilename string) error {
	if id <= 0 {
		return errors.New("invalid product ID")
	}

	// Get existing product
	existing, err := s.productRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	// Validate product data
	if err := s.validateProduct(product); err != nil {
		return err
	}

	// Check if code is being changed and if new code already exists
	if product.Code != existing.Code {
		codeExists, _ := s.productRepo.FindByCode(product.Code)
		if codeExists != nil {
			return fmt.Errorf("product with code %s already exists", product.Code)
		}
	}

	// Set ID for update
	product.ID = id

	// Handle photo update
	oldPhotoID := existing.MainPhotoID
	if newPhoto != nil {
		// Upload new photo
		photoURL, photoID, err := s.cloudinaryService.UploadProductImage(ctx, newPhoto, photoFilename)
		if err != nil {
			return fmt.Errorf("failed to upload photo: %w", err)
		}
		product.MainPhotoURL = photoURL
		product.MainPhotoID = photoID

		// Delete old photo from Cloudinary (best effort)
		if oldPhotoID != "" {
			_ = s.cloudinaryService.DeleteImage(ctx, oldPhotoID)
		}
	} else {
		// Keep existing photo
		product.MainPhotoURL = existing.MainPhotoURL
		product.MainPhotoID = existing.MainPhotoID
	}

	// Start transaction for variant updates
	tx, err := s.db.Beginx()
	if err != nil {
		// Rollback: if we uploaded a new photo, delete it
		if newPhoto != nil && product.MainPhotoID != "" {
			_ = s.cloudinaryService.DeleteImage(ctx, product.MainPhotoID)
		}
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Update product in database
	err = s.productRepo.Update(product)
	if err != nil {
		// Rollback: if we uploaded a new photo, delete it
		if newPhoto != nil && product.MainPhotoID != "" {
			_ = s.cloudinaryService.DeleteImage(ctx, product.MainPhotoID)
		}
		return fmt.Errorf("failed to update product: %w", err)
	}

	// Handle variants update
	if len(product.Variants) > 0 {
		// Get existing variants to delete their photos from Cloudinary
		existingVariants := existing.Variants
		if existingVariants == nil {
			existingVariants = []models.ProductVariant{}
		}

		// Create a set of PhotoIDs that are being preserved in new variants
		preservedPhotoIDs := make(map[string]bool)
		for _, newVariant := range product.Variants {
			if newVariant.PhotoID != "" {
				preservedPhotoIDs[newVariant.PhotoID] = true
			}
		}

		// Delete old variants
		err = s.productRepo.DeleteVariantsByProductID(tx, id)
		if err != nil {
			// Rollback: if we uploaded a new photo, delete it
			if newPhoto != nil && product.MainPhotoID != "" {
				_ = s.cloudinaryService.DeleteImage(ctx, product.MainPhotoID)
			}
			return fmt.Errorf("failed to delete old variants: %w", err)
		}

		// Delete old variant photos from Cloudinary only if they're not being preserved
		for _, oldVariant := range existingVariants {
			if oldVariant.PhotoID != "" && !preservedPhotoIDs[oldVariant.PhotoID] {
				_ = s.cloudinaryService.DeleteImage(ctx, oldVariant.PhotoID)
			}
		}

		// Validate and create new variants
		for _, variant := range product.Variants {
			if variant.Color == "" {
				// Rollback: if we uploaded a new photo, delete it
				if newPhoto != nil && product.MainPhotoID != "" {
					_ = s.cloudinaryService.DeleteImage(ctx, product.MainPhotoID)
				}
				return errors.New("variant color is required")
			}
		}

		err = s.productRepo.CreateVariants(tx, id, product.Variants)
		if err != nil {
			// Rollback: if we uploaded a new photo, delete it
			if newPhoto != nil && product.MainPhotoID != "" {
				_ = s.cloudinaryService.DeleteImage(ctx, product.MainPhotoID)
			}
			// Rollback: delete uploaded variant photos
			for _, variant := range product.Variants {
				if variant.PhotoID != "" {
					_ = s.cloudinaryService.DeleteImage(ctx, variant.PhotoID)
				}
			}
			return fmt.Errorf("failed to create variants: %w", err)
		}
	} else {
		// If no variants provided, delete all existing variants
		existingVariants := existing.Variants
		if existingVariants == nil {
			existingVariants = []models.ProductVariant{}
		}
		err = s.productRepo.DeleteVariantsByProductID(tx, id)
		if err != nil {
			return fmt.Errorf("failed to delete variants: %w", err)
		}
		// Delete old variant photos from Cloudinary (best effort)
		for _, oldVariant := range existingVariants {
			if oldVariant.PhotoID != "" {
				_ = s.cloudinaryService.DeleteImage(ctx, oldVariant.PhotoID)
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		// Rollback: if we uploaded a new photo, delete it
		if newPhoto != nil && product.MainPhotoID != "" {
			_ = s.cloudinaryService.DeleteImage(ctx, product.MainPhotoID)
		}
		// Rollback: delete uploaded variant photos
		for _, variant := range product.Variants {
			if variant.PhotoID != "" {
				_ = s.cloudinaryService.DeleteImage(ctx, variant.PhotoID)
			}
		}
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete deletes a product and its photos
func (s *ProductService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("invalid product ID")
	}

	// Get product to retrieve photo IDs
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	// Delete from database (cascades to variants)
	err = s.productRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	// Delete photos from Cloudinary (best effort, don't fail if this fails)
	if product.MainPhotoID != "" {
		_ = s.cloudinaryService.DeleteImage(ctx, product.MainPhotoID)
	}

	// Delete variant photos
	for _, variant := range product.Variants {
		if variant.PhotoID != "" {
			_ = s.cloudinaryService.DeleteImage(ctx, variant.PhotoID)
		}
	}

	return nil
}

// Search searches products by query
func (s *ProductService) Search(ctx context.Context, query string) ([]models.Product, error) {
	if query == "" {
		return []models.Product{}, nil
	}

	products, err := s.productRepo.Search(query)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	return products, nil
}

// validateProduct validates product data (no validation on code — freetext)
func (s *ProductService) validateProduct(product *models.Product) error {
	// Validate title
	if product.Title == "" {
		return errors.New("product title is required")
	}
	if len(product.Title) < 5 {
		return errors.New("product title must be at least 5 characters")
	}
	if len(product.Title) > 200 {
		return errors.New("product title must not exceed 200 characters")
	}

	// Validate base price
	if product.BasePrice < 0.01 {
		return errors.New("base price must be at least 0.01")
	}
	if product.BasePrice > 99999999.99 {
		return errors.New("base price must not exceed 99,999,999.99")
	}

	return nil
}
