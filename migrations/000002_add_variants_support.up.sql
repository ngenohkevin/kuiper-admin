-- Add variants support to products table

-- Add has_variants column
ALTER TABLE products ADD COLUMN IF NOT EXISTS has_variants BOOLEAN DEFAULT FALSE;

-- Add variants JSONB column
ALTER TABLE products ADD COLUMN IF NOT EXISTS variants JSONB DEFAULT '[]'::jsonb;

-- Add reviewer_name column to reviews table (for compatibility)
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS reviewer_name VARCHAR(255);

-- Update products that might need slug conflict resolution
-- Create a function to fix duplicate slugs
CREATE OR REPLACE FUNCTION fix_duplicate_slugs() RETURNS void AS $$
DECLARE
    rec RECORD;
    new_slug VARCHAR(255);
    counter INTEGER;
BEGIN
    -- Find products with duplicate slugs
    FOR rec IN 
        SELECT slug, COUNT(*) as count 
        FROM products 
        GROUP BY slug 
        HAVING COUNT(*) > 1
    LOOP
        -- Update all but the first product with the duplicate slug
        counter := 1;
        FOR rec IN 
            SELECT id, slug 
            FROM products 
            WHERE slug = rec.slug 
            ORDER BY created_at 
            OFFSET 1
        LOOP
            new_slug := rec.slug || '-' || counter;
            -- Ensure the new slug is also unique
            WHILE EXISTS (SELECT 1 FROM products WHERE slug = new_slug) LOOP
                counter := counter + 1;
                new_slug := rec.slug || '-' || counter;
            END LOOP;
            
            UPDATE products SET slug = new_slug WHERE id = rec.id;
            counter := counter + 1;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Execute the function to fix existing duplicates
SELECT fix_duplicate_slugs();

-- Drop the function after use
DROP FUNCTION fix_duplicate_slugs();

-- Create index on variants column for better query performance
CREATE INDEX IF NOT EXISTS idx_products_variants ON products USING GIN (variants);
