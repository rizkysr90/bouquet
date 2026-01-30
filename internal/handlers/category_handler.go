package handlers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rizkysr90/aslam-flower/internal/repositories"
	"github.com/rizkysr90/aslam-flower/internal/services"
)

// CategoryHandler handles category CRUD routes
type CategoryHandler struct {
	categoryService *services.CategoryService
	categoryRepo    *repositories.CategoryRepository
}

// NewCategoryHandler creates a new category handler
func NewCategoryHandler(
	categoryService *services.CategoryService,
	categoryRepo *repositories.CategoryRepository,
) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
		categoryRepo:    categoryRepo,
	}
}

// ListCategories renders the categories list page
func (h *CategoryHandler) ListCategories(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get all categories
	categories, err := h.categoryService.GetAll(ctx)
	if err != nil {
		return c.Status(500).SendString("Failed to load categories")
	}

	// Get product count for each category and create map for template
	type CategoryWithCount struct {
		Category     map[string]interface{}
		ProductCount int
	}
	categoriesWithCounts := make([]CategoryWithCount, 0, len(categories))

	for _, category := range categories {
		count, err := h.categoryRepo.CountProducts(category.ID)
		if err != nil {
			// If error, just set count to 0
			count = 0
		}
		catMap := map[string]interface{}{
			"ID":   category.ID,
			"Name": category.Name,
			"Slug": category.Slug,
		}
		categoriesWithCounts = append(categoriesWithCounts, CategoryWithCount{
			Category:     catMap,
			ProductCount: count,
		})
	}

	// Get success message from query
	successMsg := c.Query("success", "")

	return c.Render("pages/admin/categories", fiber.Map{
		"Title":        "Manage Categories",
		"Categories":   categoriesWithCounts,
		"Success":      successMsg,
		"CSRFToken":    getCSRFToken(c),
		"CurrentPage":  "categories",
		"ContentBlock": "admin-content-categories",
	}, "layouts/admin")
}

// NewCategoryForm renders the category creation form
func (h *CategoryHandler) NewCategoryForm(c *fiber.Ctx) error {
	return c.Render("pages/admin/category-form", fiber.Map{
		"Title":        "Add Category",
		"Category":     nil,
		"IsEdit":       false,
		"CSRFToken":    getCSRFToken(c),
		"CurrentPage":  "categories",
		"ContentBlock": "admin-content-category-form",
	}, "layouts/admin")
}

// CreateCategory handles category creation
func (h *CategoryHandler) CreateCategory(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse form data
	name := c.FormValue("name")
	if name == "" {
		return c.Render("pages/admin/category-form", fiber.Map{
			"Title":        "Add Category",
			"Category":     nil,
			"IsEdit":       false,
			"Error":        "Category name is required",
			"CSRFToken":    getCSRFToken(c),
			"CurrentPage":  "categories",
			"ContentBlock": "admin-content-category-form",
		}, "layouts/admin")
	}

	// Create category
	category, err := h.categoryService.Create(ctx, name)
	if err != nil {
		return c.Render("pages/admin/category-form", fiber.Map{
			"Title":        "Add Category",
			"Category":     nil,
			"IsEdit":       false,
			"Error":        err.Error(),
			"CSRFToken":    getCSRFToken(c),
			"CurrentPage":  "categories",
			"ContentBlock": "admin-content-category-form",
		}, "layouts/admin")
	}

	// Redirect with success message
	return c.Redirect(fmt.Sprintf("/admin/categories?success=Category '%%27%s%%27 created successfully", category.Name))
}

// EditCategoryForm renders the category edit form
func (h *CategoryHandler) EditCategoryForm(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get category ID
	idParam := c.Params("id")
	categoryID, err := strconv.Atoi(idParam)
	if err != nil || categoryID <= 0 {
		return c.Status(404).SendString("Category not found")
	}

	// Get category
	category, err := h.categoryService.GetByID(ctx, categoryID)
	if err != nil {
		return c.Status(404).SendString("Category not found")
	}

	return c.Render("pages/admin/category-form", fiber.Map{
		"Title":        "Edit Category",
		"Category":     category,
		"IsEdit":       true,
		"CSRFToken":    getCSRFToken(c),
		"CurrentPage":  "categories",
		"ContentBlock": "admin-content-category-form",
	}, "layouts/admin")
}

// UpdateCategory handles category update
func (h *CategoryHandler) UpdateCategory(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get category ID
	idParam := c.Params("id")
	categoryID, err := strconv.Atoi(idParam)
	if err != nil || categoryID <= 0 {
		return c.Status(404).SendString("Category not found")
	}

	// Parse form data
	name := c.FormValue("name")
	if name == "" {
		// Get category for re-rendering
		category, _ := h.categoryService.GetByID(ctx, categoryID)
		return c.Render("pages/admin/category-form", fiber.Map{
			"Title":        "Edit Category",
			"Category":     category,
			"IsEdit":       true,
			"Error":        "Category name is required",
			"CSRFToken":    getCSRFToken(c),
			"CurrentPage":  "categories",
			"ContentBlock": "admin-content-category-form",
		}, "layouts/admin")
	}

	// Update category
	category, err := h.categoryService.Update(ctx, categoryID, name)
	if err != nil {
		// Get category for re-rendering
		existingCategory, _ := h.categoryService.GetByID(ctx, categoryID)
		return c.Render("pages/admin/category-form", fiber.Map{
			"Title":        "Edit Category",
			"Category":     existingCategory,
			"IsEdit":       true,
			"Error":        err.Error(),
			"CSRFToken":    getCSRFToken(c),
			"CurrentPage":  "categories",
			"ContentBlock": "admin-content-category-form",
		}, "layouts/admin")
	}

	// Redirect with success message
	return c.Redirect(fmt.Sprintf("/admin/categories?success=Category '%%27%s%%27 updated successfully", category.Name))
}

// DeleteCategory handles category deletion (htmx)
func (h *CategoryHandler) DeleteCategory(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get category ID
	idParam := c.Params("id")
	categoryID, err := strconv.Atoi(idParam)
	if err != nil || categoryID <= 0 {
		return c.Status(400).SendString("Invalid category ID")
	}

	// Delete category
	err = h.categoryService.Delete(ctx, categoryID)
	if err != nil {
		// Return error message for htmx
		return c.Status(400).SendString(fmt.Sprintf("<div class='text-red-600 p-4 bg-red-50 rounded-lg'>%s</div>", err.Error()))
	}

	// Return empty response to remove row (htmx will remove the target element)
	return c.SendString("")
}
