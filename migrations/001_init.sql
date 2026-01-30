-- Migration: 001_init.sql
-- Description: Create initial database schema with all tables and indexes
-- Created: 2026-01-15

-- Categories table
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_categories_slug ON categories(slug);

-- Products table
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    main_photo_url VARCHAR(500),
    main_photo_id VARCHAR(200),      -- Cloudinary public_id (for deletion)
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    base_price DECIMAL(12,2) NOT NULL CHECK (base_price >= 0),
    is_sale BOOLEAN DEFAULT FALSE,
    is_soldout BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_products_code ON products(code);
CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_flags ON products(is_sale, is_soldout);
CREATE INDEX idx_products_search ON products USING gin(to_tsvector('indonesian', title));

-- Product variants table
CREATE TABLE product_variants (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    color VARCHAR(50) NOT NULL,
    photo_url VARCHAR(500),
    photo_id VARCHAR(200),           -- Cloudinary public_id
    price_adjustment DECIMAL(12,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, color)
);

CREATE INDEX idx_variants_product ON product_variants(product_id);

-- Admins table
CREATE TABLE admins (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_admins_username ON admins(username);

