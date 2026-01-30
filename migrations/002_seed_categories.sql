-- Migration: 002_seed_categories.sql
-- Description: Insert sample categories
-- Created: 2026-01-15

INSERT INTO categories (name, slug) VALUES
    ('Kertas Bouquet', 'kertas-bouquet'),
    ('Pita & Ribbon', 'pita-ribbon'),
    ('Aksesoris Dekorasi', 'aksesoris-dekorasi'),
    ('Wrapping Material', 'wrapping-material')
ON CONFLICT (name) DO NOTHING;

