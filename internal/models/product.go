package models

import "time"

// Product represents a product entity
type Product struct {
	ID           int       `db:"id" json:"id"`
	Code         string    `db:"code" json:"code"`
	Title        string    `db:"title" json:"title"`
	Description  string    `db:"description" json:"description"`
	MainPhotoURL string    `db:"main_photo_url" json:"main_photo_url"`
	MainPhotoID  string    `db:"main_photo_id" json:"main_photo_id"`
	CategoryID   *int      `db:"category_id" json:"category_id"`
	BasePrice    float64   `db:"base_price" json:"base_price"`
	IsSold       bool      `db:"is_sold" json:"is_sold"` // For availability filtering
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`

	// Relations (not in DB)
	Category *Category        `db:"-" json:"category,omitempty"`
	Variants []ProductVariant `db:"-" json:"variants,omitempty"`
}

// ProductVariant represents a product color variant
type ProductVariant struct {
	ID              int       `db:"id" json:"id"`
	ProductID       int       `db:"product_id" json:"product_id"`
	Color           string    `db:"color" json:"color"`
	PhotoURL        string    `db:"photo_url" json:"photo_url"`
	PhotoID         string    `db:"photo_id" json:"photo_id"`
	PriceAdjustment float64   `db:"price_adjustment" json:"price_adjustment"`
	IsSale          bool      `db:"is_sale" json:"is_sale"` // true = SALE, false = SOLD
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// FinalPrice returns base_price + price_adjustment
func (v *ProductVariant) FinalPrice(basePrice float64) float64 {
	return basePrice + v.PriceAdjustment
}
