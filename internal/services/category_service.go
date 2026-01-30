package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/rizkysr90/aslam-flower/internal/models"
	"github.com/rizkysr90/aslam-flower/internal/repositories"
	"github.com/rizkysr90/aslam-flower/internal/utils"
)

// CategoryService handles category business logic
type CategoryService struct {
	categoryRepo *repositories.CategoryRepository
}

// NewCategoryService creates a new category service
func NewCategoryService(categoryRepo *repositories.CategoryRepository) *CategoryService {
	return &CategoryService{
		categoryRepo: categoryRepo,
	}
}

// GetAll retrieves all categories
func (s *CategoryService) GetAll(ctx context.Context) ([]models.Category, error) {
	return s.categoryRepo.FindAll()
}

// GetByID retrieves a category by ID
func (s *CategoryService) GetByID(ctx context.Context, id int) (*models.Category, error) {
	if id <= 0 {
		return nil, errors.New("invalid category ID")
	}

	category, err := s.categoryRepo.FindByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("category not found")
		}
		return nil, fmt.Errorf("failed to fetch category: %w", err)
	}

	return category, nil
}

// Create creates a new category
func (s *CategoryService) Create(ctx context.Context, name string) (*models.Category, error) {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("category name is required")
	}
	if len(name) < 3 {
		return nil, errors.New("category name must be at least 3 characters")
	}
	if len(name) > 100 {
		return nil, errors.New("category name must be at most 100 characters")
	}

	// Generate slug
	slug := utils.GenerateSlug(name)
	if slug == "" {
		return nil, errors.New("invalid category name: cannot generate slug")
	}

	// Create category
	category := &models.Category{
		Name: name,
		Slug: slug,
	}

	err := s.categoryRepo.Create(category)
	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return nil, errors.New("category name already exists")
			}
		}
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return category, nil
}

// Update updates an existing category
func (s *CategoryService) Update(ctx context.Context, id int, name string) (*models.Category, error) {
	// Validate ID
	if id <= 0 {
		return nil, errors.New("invalid category ID")
	}

	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("category name is required")
	}
	if len(name) < 3 {
		return nil, errors.New("category name must be at least 3 characters")
	}
	if len(name) > 100 {
		return nil, errors.New("category name must be at most 100 characters")
	}

	// Get existing category
	existing, err := s.categoryRepo.FindByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("category not found")
		}
		return nil, fmt.Errorf("failed to fetch category: %w", err)
	}

	// Generate new slug if name changed
	slug := existing.Slug
	if name != existing.Name {
		slug = utils.GenerateSlug(name)
		if slug == "" {
			return nil, errors.New("invalid category name: cannot generate slug")
		}
	}

	// Update category
	category := &models.Category{
		ID:   id,
		Name: name,
		Slug: slug,
	}

	err = s.categoryRepo.Update(category)
	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return nil, errors.New("category name already exists")
			}
		}
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	// Fetch updated category
	updated, err := s.categoryRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated category: %w", err)
	}

	return updated, nil
}

// Delete deletes a category
func (s *CategoryService) Delete(ctx context.Context, id int) error {
	// Validate ID
	if id <= 0 {
		return errors.New("invalid category ID")
	}

	// Check if category exists
	_, err := s.categoryRepo.FindByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("category not found")
		}
		return fmt.Errorf("failed to fetch category: %w", err)
	}

	// Check if category has products
	count, err := s.categoryRepo.CountProducts(id)
	if err != nil {
		return fmt.Errorf("failed to count products: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete category with %d products", count)
	}

	// Delete category
	err = s.categoryRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}
