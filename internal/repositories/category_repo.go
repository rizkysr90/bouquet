package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/rizkysr90/aslam-flower/internal/models"
)

// CategoryRepository handles category data access
type CategoryRepository struct {
	db *sqlx.DB
}

// NewCategoryRepository creates a new category repository
func NewCategoryRepository(db *sqlx.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// FindAll retrieves all categories
func (r *CategoryRepository) FindAll() ([]models.Category, error) {
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM categories
		ORDER BY name ASC
	`

	var categories []models.Category
	err := r.db.Select(&categories, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}

	return categories, nil
}

// FindByID retrieves a category by ID
func (r *CategoryRepository) FindByID(id int) (*models.Category, error) {
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM categories
		WHERE id = $1
	`

	var category models.Category
	err := r.db.Get(&category, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch category: %w", err)
	}

	return &category, nil
}

// FindBySlug retrieves a category by slug
func (r *CategoryRepository) FindBySlug(slug string) (*models.Category, error) {
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM categories
		WHERE slug = $1
	`

	var category models.Category
	err := r.db.Get(&category, query, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch category: %w", err)
	}

	return &category, nil
}

// Create inserts a new category
func (r *CategoryRepository) Create(category *models.Category) error {
	query := `
		INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		category.Name,
		category.Slug,
	).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}

	return nil
}

// Update updates an existing category
func (r *CategoryRepository) Update(category *models.Category) error {
	query := `
		UPDATE categories
		SET 
			name = $1,
			slug = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		category.Name,
		category.Slug,
		category.ID,
	).Scan(&category.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	return nil
}

// Delete removes a category by ID
func (r *CategoryRepository) Delete(id int) error {
	query := `DELETE FROM categories WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("category with id %d not found", id)
	}

	return nil
}

// CountProducts counts how many products belong to a category
func (r *CategoryRepository) CountProducts(categoryID int) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM products 
		WHERE category_id = $1
	`

	var count int
	err := r.db.Get(&count, query, categoryID)
	if err != nil {
		return 0, fmt.Errorf("failed to count products: %w", err)
	}

	return count, nil
}
