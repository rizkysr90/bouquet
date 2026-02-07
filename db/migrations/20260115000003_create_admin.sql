-- migrate:up
INSERT INTO admins (username, password_hash) VALUES
    ('admin', '$2a$10$68BHyucTsst8/oDDLnRVA.V0p0vqG781mG0Wngc/1qcuvws7jO.S6')
ON CONFLICT (username) DO NOTHING;

-- migrate:down
DELETE FROM admins WHERE username = 'admin';
