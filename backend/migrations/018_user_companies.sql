-- 018_user_companies.sql: many-to-many user ↔ company assignments

CREATE TABLE user_companies (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, company_id)
);

CREATE INDEX idx_user_companies_company ON user_companies(company_id);

-- Migrate existing single-company assignments (non-superadmin only).
INSERT INTO user_companies (user_id, company_id)
SELECT id, company_id
FROM users
WHERE company_id IS NOT NULL AND role != 'superadmin';

-- company_id on users is deprecated; junction table is the source of truth.
UPDATE users SET company_id = NULL;
