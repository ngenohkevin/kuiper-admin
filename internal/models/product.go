package models

import (
	"context"
	"crypto/md5"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/kuiper_admin/internal/database"
)

type Product struct {
	ID           string           `json:"id"`
	CategoryID   *string          `json:"category_id"`
	Name         string           `json:"name"`
	Slug         string           `json:"slug"`
	Description  string           `json:"description"`
	Price        float64          `json:"price"`
	ImageURLs    []string         `json:"image_urls"`
	StockCount   int              `json:"stock_count"`
	IsAvailable  bool             `json:"is_available"`
	HasVariants  bool             `json:"has_variants"`
	CreatedAt    pgtype.Timestamp `json:"created_at"`
	UpdatedAt    pgtype.Timestamp `json:"updated_at"`
	Category     *Category        `json:"category,omitempty"`
	Variants     []ProductVariant `json:"variants,omitempty"`
	VariantsJSON string           `json:"variants_json,omitempty"`
}

// StringArray is a custom type for handling string arrays from Postgres
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *StringArray) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &a)
}

// PaginatedResult holds paginated data with metadata
type PaginatedResult[T any] struct {
	Data       []T   `json:"data"`
	TotalCount int64 `json:"total_count"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// GetAllProducts retrieves all products from the database (deprecated - use GetProductsPaginated)
func GetAllProducts(db *database.DB) ([]Product, error) {
	// Use pagination with a large page size for backward compatibility
	result, err := GetProductsPaginated(db, 1, 1000, "", "")
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// generateCacheKey creates a cache key for the query parameters
func generateCacheKey(page, pageSize int, categoryID, search string) string {
	key := fmt.Sprintf("products:page=%d:size=%d:cat=%s:search=%s", page, pageSize, categoryID, search)
	hash := fmt.Sprintf("%x", md5.Sum([]byte(key)))
	return "products:" + hash
}

// GetProductsPaginated retrieves products with pagination and optional filtering
func GetProductsPaginated(db *database.DB, page, pageSize int, categoryID, search string) (*PaginatedResult[Product], error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20 // Default page size
	}
	if pageSize > 100 {
		pageSize = 100 // Maximum page size
	}

	// Check cache first (cache for 5 minutes for frequently accessed data)
	cacheKey := generateCacheKey(page, pageSize, categoryID, search)
	if cached, found := db.Cache.Get(cacheKey); found {
		if result, ok := cached.(*PaginatedResult[Product]); ok {
			return result, nil
		}
	}

	offset := (page - 1) * pageSize

	// Build WHERE conditions
	var whereConditions []string
	var args []interface{}
	argIndex := 1

	if categoryID != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("p.category_id = $%d", argIndex))
		args = append(args, categoryID)
		argIndex++
	}

	if search != "" {
		// Use ILIKE for search (fallback for compatibility)
		whereConditions = append(whereConditions, fmt.Sprintf("(LOWER(p.name) ILIKE LOWER($%d) OR LOWER(p.description) ILIKE LOWER($%d))", argIndex, argIndex))
		args = append(args, "%"+search+"%")
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + fmt.Sprintf("(%s)", fmt.Sprintf("%s", whereConditions[0]))
		for i := 1; i < len(whereConditions); i++ {
			whereClause += fmt.Sprintf(" AND (%s)", whereConditions[i])
		}
	}

	// Count total records - simplified without JOIN
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM products p
		%s
	`, whereClause)

	var totalCount int64
	err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("error counting products: %w", err)
	}

	// Get paginated data - simplified without category JOIN for performance
	query := fmt.Sprintf(`
		SELECT p.id, p.category_id, p.name, p.slug, p.description, 
		       p.price, p.image_urls, p.stock_count, p.is_available, p.has_variants,
		       p.created_at, p.updated_at, p.variants
		FROM products p
		%s
		ORDER BY p.created_at DESC, p.name
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		var variantsJSON []byte

		if err := rows.Scan(
			&p.ID, &p.CategoryID, &p.Name, &p.Slug, &p.Description,
			&p.Price, &p.ImageURLs, &p.StockCount, &p.IsAvailable, &p.HasVariants,
			&p.CreatedAt, &p.UpdatedAt, &variantsJSON,
		); err != nil {
			return nil, fmt.Errorf("error scanning product row: %w", err)
		}

		// Load category separately if needed for better performance
		if p.CategoryID != nil && *p.CategoryID != "" {
			// We can optionally load category data here if the template requires it
			// For now, we'll skip it to improve performance
		}

		// Parse variants from JSONB
		if variantsJSON != nil && string(variantsJSON) != "[]" && string(variantsJSON) != "null" {
			p.VariantsJSON = string(variantsJSON)
			var variants []ProductVariant
			if err := json.Unmarshal(variantsJSON, &variants); err != nil {
				log.Printf("Error parsing variants JSON: %v", err)
			} else {
				// Set ProductID for each variant
				for i := range variants {
					variants[i].ProductID = p.ID
					// If weight field is populated, use it as the name
					if variants[i].Weight != "" && variants[i].Name == "" {
						variants[i].Name = variants[i].Weight
					}
				}
				p.Variants = variants
			}
		}

		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	hasNext := page < totalPages
	hasPrev := page > 1

	result := &PaginatedResult[Product]{
		Data:       products,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}

	// Cache the result for 5 minutes
	db.Cache.Set(cacheKey, result, 5*time.Minute)

	return result, nil
}

// GetProductByID retrieves a single product by ID
func GetProductByID(db *database.DB, id string) (Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT p.id, p.category_id, p.name, p.slug, p.description, 
		       p.price, p.image_urls, p.stock_count, p.is_available, p.has_variants,
		       p.created_at, p.updated_at, p.variants,
		       c.id, c.name, c.slug, c.parent_id, c.created_at
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.id = $1
	`

	var p Product
	var variantsJSON []byte
	// Use nullable types for category fields to handle LEFT JOIN NULLs
	var catID, catName, catSlug, catParentID *string
	var catCreatedAt *time.Time

	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.CategoryID, &p.Name, &p.Slug, &p.Description,
		&p.Price, &p.ImageURLs, &p.StockCount, &p.IsAvailable, &p.HasVariants,
		&p.CreatedAt, &p.UpdatedAt, &variantsJSON,
		&catID, &catName, &catSlug, &catParentID, &catCreatedAt,
	)
	if err != nil {
		return Product{}, fmt.Errorf("error finding product: %w", err)
	}

	// Only create Category if we have valid category data
	if catID != nil && *catID != "" {
		c := Category{
			ID:   *catID,
			Name: *catName,
			Slug: *catSlug,
		}
		if catParentID != nil {
			c.ParentID = catParentID
		}
		if catCreatedAt != nil {
			c.CreatedAt = pgtype.Timestamp{Time: *catCreatedAt, Valid: true}
		}
		p.Category = &c
	}

	// Parse variants from JSONB
	if variantsJSON != nil && string(variantsJSON) != "[]" && string(variantsJSON) != "null" {
		p.VariantsJSON = string(variantsJSON)
		var variants []ProductVariant
		if err := json.Unmarshal(variantsJSON, &variants); err != nil {
			log.Printf("Error parsing variants JSON: %v", err)
		} else {
			// Set ProductID for each variant
			for i := range variants {
				variants[i].ProductID = p.ID
				// If weight field is populated, use it as the name
				if variants[i].Weight != "" && variants[i].Name == "" {
					variants[i].Name = variants[i].Weight
				}
			}
			p.Variants = variants
		}
	}

	return p, nil
}

// CreateProduct creates a new product in the database
func CreateProduct(db *database.DB, categoryID *string, name, slug, description string,
	price float64, imageURLs []string, stockCount int, isAvailable bool, hasVariants bool) (Product, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Generate a UUID for the new product
	newID := uuid.New().String()

	// Log the input parameters for debugging
	catIDValue := "<nil>"
	if categoryID != nil {
		catIDValue = *categoryID
	}
	log.Printf("Creating product with id=%s, name=%s, slug=%s, category_id=%s, price=%.2f, stockCount=%d, isAvailable=%v, hasVariants=%v",
		newID, name, slug, catIDValue, price, stockCount, isAvailable, hasVariants)
	log.Printf("Image URLs: %v", imageURLs)

	query := `
		INSERT INTO products (id, category_id, name, slug, description, price, image_urls, stock_count, is_available, has_variants, created_at, updated_at, variants)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, '[]'::jsonb)
		RETURNING id, category_id, name, slug, description, price, image_urls, stock_count, is_available, has_variants, created_at, updated_at, variants
	`

	var p Product
	var variantsJSON []byte

	err := db.Pool.QueryRow(ctx, query, newID, categoryID, name, slug, description, price, imageURLs, stockCount, isAvailable, hasVariants).Scan(
		&p.ID, &p.CategoryID, &p.Name, &p.Slug, &p.Description,
		&p.Price, &p.ImageURLs, &p.StockCount, &p.IsAvailable, &p.HasVariants,
		&p.CreatedAt, &p.UpdatedAt, &variantsJSON,
	)
	if err != nil {
		log.Printf("Database error creating product: %v", err)
		return Product{}, fmt.Errorf("error creating product: %w", err)
	}

	log.Printf("Successfully created product with ID: %s", p.ID)
	return p, nil
}

// UpdateProduct updates an existing product in the database
func UpdateProduct(db *database.DB, id string, categoryID *string, name, slug, description string,
	price float64, imageURLs []string, stockCount int, isAvailable bool, hasVariants bool) (Product, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get current product to preserve variants if not changing has_variants from true to false
	var currentVariantsJSON []byte
	if !hasVariants {
		// If we're setting has_variants to false, check if there are variants we need to keep
		err := db.Pool.QueryRow(ctx, "SELECT has_variants, variants FROM products WHERE id = $1", id).Scan(
			&hasVariants, &currentVariantsJSON)
		if err != nil {
			log.Printf("Error getting current product variants: %v", err)
		}
	}

	query := `
		UPDATE products
		SET category_id = $2, name = $3, slug = $4, description = $5, 
			price = $6, image_urls = $7, stock_count = $8, is_available = $9, has_variants = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id, category_id, name, slug, description, price, image_urls, stock_count, is_available, has_variants, created_at, updated_at, variants
	`

	var p Product
	var variantsJSON []byte

	err := db.Pool.QueryRow(ctx, query, id, categoryID, name, slug, description, price, imageURLs, stockCount, isAvailable, hasVariants).Scan(
		&p.ID, &p.CategoryID, &p.Name, &p.Slug, &p.Description,
		&p.Price, &p.ImageURLs, &p.StockCount, &p.IsAvailable, &p.HasVariants,
		&p.CreatedAt, &p.UpdatedAt, &variantsJSON,
	)
	if err != nil {
		return Product{}, fmt.Errorf("error updating product: %w", err)
	}

	// Parse variants from JSONB
	if variantsJSON != nil && string(variantsJSON) != "[]" && string(variantsJSON) != "null" {
		p.VariantsJSON = string(variantsJSON)
		var variants []ProductVariant
		if err := json.Unmarshal(variantsJSON, &variants); err != nil {
			log.Printf("Error parsing variants JSON: %v", err)
		} else {
			// Set ProductID for each variant
			for i := range variants {
				variants[i].ProductID = p.ID
				// If weight field is populated, use it as the name
				if variants[i].Weight != "" && variants[i].Name == "" {
					variants[i].Name = variants[i].Weight
				}
			}
			p.Variants = variants
		}
	}

	return p, nil
}

// DeleteProduct deletes a product from the database
func DeleteProduct(db *database.DB, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `DELETE FROM products WHERE id = $1`

	_, err := db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error deleting product: %w", err)
	}

	return nil
}

// UpdateProductHasVariants updates the has_variants flag on a product
func UpdateProductHasVariants(db *database.DB, id string, hasVariants bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		UPDATE products
		SET has_variants = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := db.Pool.Exec(ctx, query, id, hasVariants)
	if err != nil {
		return fmt.Errorf("error updating product has_variants flag: %w", err)
	}

	return nil
}
