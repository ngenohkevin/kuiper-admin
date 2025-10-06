-- Remove variants support from products table

-- Drop index on variants column
DROP INDEX IF EXISTS idx_products_variants;

-- Remove variants column
ALTER TABLE products DROP COLUMN IF EXISTS variants;

-- Remove has_variants column
ALTER TABLE products DROP COLUMN IF EXISTS has_variants;

-- Remove reviewer_name column from reviews table
ALTER TABLE reviews DROP COLUMN IF EXISTS reviewer_name;
