-- Manual fix for missing columns and slug issues

-- First, let's add the missing columns if they don't exist
ALTER TABLE products ADD COLUMN IF NOT EXISTS has_variants BOOLEAN DEFAULT FALSE;
ALTER TABLE products ADD COLUMN IF NOT EXISTS variants JSONB DEFAULT '[]'::jsonb;

-- Add reviewer_name column to reviews if it doesn't exist
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS reviewer_name VARCHAR(255);

-- Fix duplicate slugs by adding a suffix to duplicates
WITH duplicate_slugs AS (
  SELECT slug, array_agg(id ORDER BY created_at) as ids
  FROM products
  GROUP BY slug
  HAVING COUNT(*) > 1
)
UPDATE products 
SET slug = products.slug || '-' || (row_number() OVER (PARTITION BY products.slug ORDER BY products.created_at) - 1)
FROM duplicate_slugs
WHERE products.id = ANY(duplicate_slugs.ids[2:]);

-- Create index on variants column for better performance
CREATE INDEX IF NOT EXISTS idx_products_variants ON products USING GIN (variants);
