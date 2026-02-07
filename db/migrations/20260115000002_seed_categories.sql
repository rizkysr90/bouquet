-- migrate:up
INSERT INTO categories (name, slug) VALUES
    ('Kertas Bouquet', 'kertas-bouquet'),
    ('Pita & Ribbon', 'pita-ribbon'),
    ('Aksesoris Dekorasi', 'aksesoris-dekorasi'),
    ('Wrapping Material', 'wrapping-material')
ON CONFLICT (name) DO NOTHING;

-- migrate:down
DELETE FROM categories WHERE slug IN (
    'kertas-bouquet', 'pita-ribbon', 'aksesoris-dekorasi', 'wrapping-material'
);
