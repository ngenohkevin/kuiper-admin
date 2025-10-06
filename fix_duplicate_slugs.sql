-- Simple fix for duplicate slugs
-- First, find and fix duplicate slugs one by one

-- Create a function to fix duplicates
CREATE OR REPLACE FUNCTION fix_duplicate_slugs_simple() RETURNS void AS $$
DECLARE
    rec RECORD;
    counter INTEGER;
    new_slug TEXT;
BEGIN
    -- Get all products with duplicate slugs except the first one
    FOR rec IN 
        SELECT p1.id, p1.slug
        FROM products p1
        WHERE EXISTS (
            SELECT 1 FROM products p2 
            WHERE p2.slug = p1.slug 
            AND p2.created_at < p1.created_at
        )
        ORDER BY p1.slug, p1.created_at
    LOOP
        counter := 1;
        new_slug := rec.slug || '-' || counter;
        
        -- Find a unique slug
        WHILE EXISTS (SELECT 1 FROM products WHERE slug = new_slug) LOOP
            counter := counter + 1;
            new_slug := rec.slug || '-' || counter;
        END LOOP;
        
        -- Update the product with the new unique slug
        UPDATE products SET slug = new_slug WHERE id = rec.id;
        RAISE NOTICE 'Updated product % from slug % to %', rec.id, rec.slug, new_slug;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Execute the function
SELECT fix_duplicate_slugs_simple();

-- Drop the function
DROP FUNCTION fix_duplicate_slugs_simple();
