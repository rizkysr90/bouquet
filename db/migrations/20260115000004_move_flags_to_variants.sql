-- migrate:up
ALTER TABLE product_variants
ADD COLUMN IF NOT EXISTS is_sale BOOLEAN DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_variants_sale ON product_variants(is_sale);

ALTER TABLE products
DROP COLUMN IF EXISTS is_sale;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'products' AND column_name = 'is_soldout') THEN
        ALTER TABLE products RENAME COLUMN is_soldout TO is_sold;
    END IF;
END $$;

ALTER TABLE products
ADD COLUMN IF NOT EXISTS is_sold BOOLEAN DEFAULT FALSE;

DROP INDEX IF EXISTS idx_products_flags;
CREATE INDEX IF NOT EXISTS idx_products_sold ON products(is_sold);

-- migrate:down
-- Rollback: re-add is_sale to products, remove from variants; rename is_sold back to is_soldout
ALTER TABLE product_variants DROP COLUMN IF EXISTS is_sale;
DROP INDEX IF EXISTS idx_variants_sale;

ALTER TABLE products ADD COLUMN IF NOT EXISTS is_sale BOOLEAN DEFAULT FALSE;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'products' AND column_name = 'is_sold') THEN
        ALTER TABLE products RENAME COLUMN is_sold TO is_soldout;
    END IF;
END $$;
DROP INDEX IF EXISTS idx_products_sold;
CREATE INDEX IF NOT EXISTS idx_products_flags ON products(is_sale, is_soldout);
