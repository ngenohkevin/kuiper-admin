-- Drop everything created in the up migration

-- Drop indices
DROP INDEX IF EXISTS idx_reviews_session_id;
DROP INDEX IF EXISTS idx_reviews_product_id;
DROP INDEX IF EXISTS idx_products_category_id;
DROP INDEX IF EXISTS idx_categories_parent_id;

-- Drop tables
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
