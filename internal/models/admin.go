package models

import "time"

// Admin represents an admin user
type Admin struct {
	ID           int       `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"` // Never expose in JSON
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
