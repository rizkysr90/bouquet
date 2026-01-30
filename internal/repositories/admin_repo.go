package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/rizkysr90/aslam-flower/internal/models"
)

// AdminRepository handles admin data access
type AdminRepository struct {
	db *sqlx.DB
}

// NewAdminRepository creates a new admin repository
func NewAdminRepository(db *sqlx.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// FindByUsername retrieves an admin by username
func (r *AdminRepository) FindByUsername(username string) (*models.Admin, error) {
	query := `
		SELECT id, username, password_hash, created_at
		FROM admins
		WHERE username = $1
	`

	var admin models.Admin
	err := r.db.Get(&admin, query, username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch admin: %w", err)
	}

	return &admin, nil
}

// Create inserts a new admin
func (r *AdminRepository) Create(admin *models.Admin) error {
	query := `
		INSERT INTO admins (username, password_hash)
		VALUES ($1, $2)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(
		query,
		admin.Username,
		admin.PasswordHash,
	).Scan(&admin.ID, &admin.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}

	return nil
}
