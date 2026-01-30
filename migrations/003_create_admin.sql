-- Migration: 003_create_admin.sql
-- Description: Create default admin user (username: admin, password: admin123)
-- Password hash generated with bcrypt cost factor 10
-- Created: 2026-01-15

INSERT INTO admins (username, password_hash) VALUES
    ('admin', '$2a$10$68BHyucTsst8/oDDLnRVA.V0p0vqG781mG0Wngc/1qcuvws7jO.S6')
ON CONFLICT (username) DO NOTHING;

