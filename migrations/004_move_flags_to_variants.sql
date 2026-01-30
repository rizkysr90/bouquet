-- Migration: 004_move_flags_to_variants.sql
-- Description: Move is_sale flag from products to product_variants, keep is_sold for availability filtering
-- Created: 2026-01-15

-- Add is_sale flag to product_variants
ALTER TABLE product_variants
ADD COLUMN is_sale BOOLEAN DEFAULT FALSE;

-- Create index for variant flags
CREATE INDEX idx_variants_sale ON product_variants(is_sale);

-- Remove is_sale from products table (keep is_soldout as is_sold for availability filtering)
ALTER TABLE products
DROP COLUMN IF EXISTS is_sale;

-- Rename is_soldout to is_sold for clarity (if column exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'products' AND column_name = 'is_soldout') THEN
        ALTER TABLE products RENAME COLUMN is_soldout TO is_sold;
    END IF;
END $$;

-- Add is_sold column if it doesn't exist (for fresh databases)
ALTER TABLE products
ADD COLUMN IF NOT EXISTS is_sold BOOLEAN DEFAULT FALSE;

-- Update index on products flags
DROP INDEX IF EXISTS idx_products_flags;
CREATE INDEX IF NOT EXISTS idx_products_sold ON products(is_sold);

