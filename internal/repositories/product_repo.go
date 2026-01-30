package repositories

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rizkysr90/aslam-flower/internal/models"
)

// ProductRepository handles product data access
type ProductRepository struct {
	db *sqlx.DB
}

// NewProductRepository creates a new product repository
func NewProductRepository(db *sqlx.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// ProductFilters contains filtering options for products
type ProductFilters struct {
	CategoryID  *int
	MinPrice    *float64
	MaxPrice    *float64
	IsSale      *bool // Filter by variant is_sale flag (true = SALE, false = SOLD)
	IsSold      *bool // Filter by product is_sold flag (for availability filtering)
	SearchQuery string
	SortBy      string // "newest", "price_asc", "price_desc", "name_asc"
	Page        int
	PageSize    int
}

// ProductListResult contains paginated product results
type ProductListResult struct {
	Products   []models.Product
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// FindAll retrieves products with filtering, sorting, and pagination
func (r *ProductRepository) FindAll(filters ProductFilters) (*ProductListResult, error) {
	// Build WHERE clause with parameterized queries
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	// Category filter
	if filters.CategoryID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("p.category_id = $%d", argIndex))
		args = append(args, *filters.CategoryID)
		argIndex++
	}

	// Price range filters
	if filters.MinPrice != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("p.base_price >= $%d", argIndex))
		args = append(args, *filters.MinPrice)
		argIndex++
	}
	if filters.MaxPrice != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("p.base_price <= $%d", argIndex))
		args = append(args, *filters.MaxPrice)
		argIndex++
	}

	// Sale filter - check if product has at least one variant with matching is_sale flag
	if filters.IsSale != nil {
		whereConditions = append(whereConditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM product_variants pv 
				WHERE pv.product_id = p.id AND pv.is_sale = $%d
			)
		`, argIndex))
		args = append(args, *filters.IsSale)
		argIndex++
	}

	// Sold filter - filter by product is_sold flag (for availability)
	if filters.IsSold != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("p.is_sold = $%d", argIndex))
		args = append(args, *filters.IsSold)
		argIndex++
	}

	// Search filter (by title or code)
	if filters.SearchQuery != "" {
		searchPattern := "%" + filters.SearchQuery + "%"
		whereConditions = append(whereConditions, fmt.Sprintf("(p.title ILIKE $%d OR p.code ILIKE $%d)", argIndex, argIndex))
		args = append(args, searchPattern)
		argIndex++
	}

	// Build WHERE clause
	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Build ORDER BY clause
	orderBy := "p.created_at DESC" // default: newest first
	switch filters.SortBy {
	case "price_asc":
		orderBy = "p.base_price ASC"
	case "price_desc":
		orderBy = "p.base_price DESC"
	case "name_asc":
		orderBy = "p.title ASC"
	case "newest":
		orderBy = "p.created_at DESC"
	}

	// Set defaults for pagination
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20 // default page size
	}
	offset := (filters.Page - 1) * filters.PageSize

	// Count total products matching filters
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products p %s", whereClause)
	var total int
	err := r.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count products: %w", err)
	}

	// Calculate total pages
	totalPages := (total + filters.PageSize - 1) / filters.PageSize

	// Build main query with pagination
	query := fmt.Sprintf(`
		SELECT 
			p.id, p.code, p.title, p.description, 
			p.main_photo_url, p.main_photo_id, 
			p.category_id, p.base_price, p.is_sold,
			p.created_at, p.updated_at
		FROM products p
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIndex, argIndex+1)

	args = append(args, filters.PageSize, offset)

	var products []models.Product
	err = r.db.Select(&products, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}

	// Load variants for each product
	for i := range products {
		variants, err := r.findVariantsByProductID(products[i].ID)
		if err == nil {
			if variants == nil {
				products[i].Variants = []models.ProductVariant{}
			} else {
				products[i].Variants = variants
			}
		}
	}

	return &ProductListResult{
		Products:   products,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// FindByID retrieves a product by ID with its variants
func (r *ProductRepository) FindByID(id int) (*models.Product, error) {
	query := `
		SELECT 
			id, code, title, description, 
			main_photo_url, main_photo_id, 
			category_id, base_price, is_sold,
			created_at, updated_at
		FROM products
		WHERE id = $1
	`

	var product models.Product
	err := r.db.Get(&product, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product: %w", err)
	}

	// Load variants
	variants, err := r.findVariantsByProductID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch variants: %w", err)
	}
	// Ensure Variants is never nil (use empty slice if no variants)
	if variants == nil {
		product.Variants = []models.ProductVariant{}
	} else {
		product.Variants = variants
	}

	return &product, nil
}

// FindByCode retrieves a product by code with its variants
func (r *ProductRepository) FindByCode(code string) (*models.Product, error) {
	query := `
		SELECT 
			id, code, title, description, 
			main_photo_url, main_photo_id, 
			category_id, base_price, is_sold,
			created_at, updated_at
		FROM products
		WHERE code = $1
	`

	var product models.Product
	err := r.db.Get(&product, query, code)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product: %w", err)
	}

	// Load variants
	variants, err := r.findVariantsByProductID(product.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch variants: %w", err)
	}
	// Ensure Variants is never nil (use empty slice if no variants)
	if variants == nil {
		product.Variants = []models.ProductVariant{}
	} else {
		product.Variants = variants
	}

	return &product, nil
}

// Search searches products by title or code
func (r *ProductRepository) Search(query string) ([]models.Product, error) {
	searchPattern := "%" + query + "%"
	sqlQuery := `
		SELECT 
			id, code, title, description, 
			main_photo_url, main_photo_id, 
			category_id, base_price, is_sold,
			created_at, updated_at
		FROM products
		WHERE title ILIKE $1 OR code ILIKE $1
		ORDER BY title ASC
		LIMIT 50
	`

	var products []models.Product
	err := r.db.Select(&products, sqlQuery, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	return products, nil
}

// Create inserts a new product
func (r *ProductRepository) Create(product *models.Product) error {
	query := `
		INSERT INTO products (
			code, title, description, main_photo_url, main_photo_id,
			category_id, base_price, is_sold
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		product.Code,
		product.Title,
		product.Description,
		product.MainPhotoURL,
		product.MainPhotoID,
		product.CategoryID,
		product.BasePrice,
		product.IsSold,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

// Update updates an existing product
func (r *ProductRepository) Update(product *models.Product) error {
	query := `
		UPDATE products
		SET 
			code = $1,
			title = $2,
			description = $3,
			main_photo_url = $4,
			main_photo_id = $5,
			category_id = $6,
			base_price = $7,
			is_sold = $8,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $9
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		product.Code,
		product.Title,
		product.Description,
		product.MainPhotoURL,
		product.MainPhotoID,
		product.CategoryID,
		product.BasePrice,
		product.IsSold,
		product.ID,
	).Scan(&product.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	return nil
}

// Delete removes a product by ID (cascades to variants)
func (r *ProductRepository) Delete(id int) error {
	query := `DELETE FROM products WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("product with id %d not found", id)
	}

	return nil
}

// CreateVariants creates product variants within a transaction
func (r *ProductRepository) CreateVariants(tx *sqlx.Tx, productID int, variants []models.ProductVariant) error {
	query := `
		INSERT INTO product_variants (
			product_id, color, photo_url, photo_id, price_adjustment, is_sale
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	for _, variant := range variants {
		var variantID int
		var createdAt, updatedAt time.Time

		err := tx.QueryRow(
			query,
			productID,
			variant.Color,
			variant.PhotoURL,
			variant.PhotoID,
			variant.PriceAdjustment,
			variant.IsSale,
		).Scan(&variantID, &createdAt, &updatedAt)

		if err != nil {
			return fmt.Errorf("failed to create variant %s: %w", variant.Color, err)
		}
	}

	return nil
}

// DeleteVariantsByProductID deletes all variants for a product
func (r *ProductRepository) DeleteVariantsByProductID(tx *sqlx.Tx, productID int) error {
	query := `DELETE FROM product_variants WHERE product_id = $1`
	_, err := tx.Exec(query, productID)
	if err != nil {
		return fmt.Errorf("failed to delete variants: %w", err)
	}
	return nil
}

// findVariantsByProductID is a helper to load variants for a product
func (r *ProductRepository) findVariantsByProductID(productID int) ([]models.ProductVariant, error) {
	query := `
		SELECT 
			id, product_id, color, photo_url, photo_id,
			price_adjustment, is_sale, created_at, updated_at
		FROM product_variants
		WHERE product_id = $1
		ORDER BY color ASC
	`

	var variants []models.ProductVariant
	err := r.db.Select(&variants, query, productID)
	if err != nil {
		return []models.ProductVariant{}, fmt.Errorf("failed to fetch variants: %w", err)
	}

	// Ensure we always return a non-nil slice
	if variants == nil {
		return []models.ProductVariant{}, nil
	}

	return variants, nil
}
