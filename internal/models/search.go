package models

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ngenohkevin/kuiper_admin/internal/database"
)

// SearchCategories searches for categories matching the query
func SearchCategories(db *database.DB, query string) ([]Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create a search pattern that matches the beginning of words
	searchPattern := "%" + strings.ToLower(query) + "%"

	sqlQuery := `
		SELECT id, name, slug, parent_id, created_at
		FROM categories
		WHERE LOWER(name) LIKE $1 OR LOWER(slug) LIKE $1
		ORDER BY name
	`

	rows, err := db.Pool.Query(ctx, sqlQuery, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("error searching categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning category row: %w", err)
		}
		categories = append(categories, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}

	return categories, nil
}

// SearchProducts searches for products matching the query
func SearchProducts(db *database.DB, query string) ([]Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create a search pattern
	searchPattern := "%" + strings.ToLower(query) + "%"

	sqlQuery := `
		SELECT p.id, p.category_id, p.name, p.slug, p.description, 
		       p.price, p.image_urls, p.stock_count, p.is_available, p.has_variants,
		       p.created_at, p.updated_at,
		       c.id, c.name, c.slug, c.parent_id, c.created_at
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE LOWER(p.name) LIKE $1 
		   OR LOWER(p.slug) LIKE $1 
		   OR LOWER(p.description) LIKE $1
		ORDER BY p.name
	`

	rows, err := db.Pool.Query(ctx, sqlQuery, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("error searching products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		var c Category

		if err := rows.Scan(
			&p.ID, &p.CategoryID, &p.Name, &p.Slug, &p.Description,
			&p.Price, &p.ImageURLs, &p.StockCount, &p.IsAvailable, &p.HasVariants,
			&p.CreatedAt, &p.UpdatedAt,
			&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning product row: %w", err)
		}

		if c.ID != "" {
			p.Category = &c
		}

		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// SearchReviews searches for reviews matching the query
func SearchReviews(db *database.DB, query string) ([]Review, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create a search pattern
	searchPattern := "%" + strings.ToLower(query) + "%"

	sqlQuery := `
		SELECT r.id, r.product_id, r.session_id, r.rating, r.comment, r.created_at, r.reviewer_name,
		       p.id, p.name, p.slug
		FROM reviews r
		LEFT JOIN products p ON r.product_id = p.id
		WHERE LOWER(r.comment) LIKE $1
		   OR LOWER(p.name) LIKE $1
		   OR LOWER(r.reviewer_name) LIKE $1
		ORDER BY r.created_at DESC
	`

	log.Printf("Executing search SQL query: %s with pattern: %s", sqlQuery, searchPattern)
	rows, err := db.Pool.Query(ctx, sqlQuery, searchPattern)
	if err != nil {
		log.Printf("Database search error: %v", err)
		return nil, fmt.Errorf("error searching reviews: %w", err)
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		var productID, productName, productSlug string

		if err := rows.Scan(
			&r.ID, &r.ProductID, &r.SessionID, &r.Rating, &r.Comment, &r.CreatedAt, &r.ReviewerName,
			&productID, &productName, &productSlug,
		); err != nil {
			log.Printf("Search scan error: %v", err)
			return nil, fmt.Errorf("error scanning review row: %w", err)
		}

		if productID != "" {
			r.Product = &Product{
				ID:   productID,
				Name: productName,
				Slug: productSlug,
			}
		}

		reviews = append(reviews, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating review rows: %w", err)
	}

	return reviews, nil
}
