package models

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/kuiper_admin/internal/database"
)

type Category struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Slug      string           `json:"slug"`
	ParentID  *string          `json:"parent_id"`
	CreatedAt pgtype.Timestamp `json:"created_at"`
}

// GetAllCategories retrieves all categories from the database
func GetAllCategories(db *database.DB) ([]Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, name, slug, parent_id, created_at
		FROM categories
		ORDER BY name
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying categories: %w", err)
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

// GetCategoryByID retrieves a single category by ID
func GetCategoryByID(db *database.DB, id string) (Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, name, slug, parent_id, created_at
		FROM categories
		WHERE id = $1
	`

	var c Category
	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt,
	)
	if err != nil {
		return Category{}, fmt.Errorf("error finding category: %w", err)
	}

	return c, nil
}

// CreateCategory creates a new category in the database
func CreateCategory(db *database.DB, name, slug string, parentID *string) (Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Generate a UUID for the new category
	newID := uuid.New().String()

	query := `
		INSERT INTO categories (id, name, slug, parent_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, slug, parent_id, created_at
	`

	// Log the query parameters for debugging
	parentIDValue := "<nil>"
	if parentID != nil {
		parentIDValue = *parentID
	}
	log.Printf("Creating category with id=%s, name=%s, slug=%s, parent_id=%s", newID, name, slug, parentIDValue)

	var c Category
	err := db.Pool.QueryRow(ctx, query, newID, name, slug, parentID).Scan(
		&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt,
	)
	if err != nil {
		log.Printf("Database error creating category: %v", err)
		return Category{}, fmt.Errorf("error creating category: %w", err)
	}

	log.Printf("Successfully created category with ID: %s", c.ID)
	return c, nil
}

// UpdateCategory updates an existing category in the database
func UpdateCategory(db *database.DB, id, name, slug string, parentID *string) (Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		UPDATE categories
		SET name = $2, slug = $3, parent_id = $4
		WHERE id = $1
		RETURNING id, name, slug, parent_id, created_at
	`

	var c Category
	err := db.Pool.QueryRow(ctx, query, id, name, slug, parentID).Scan(
		&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt,
	)
	if err != nil {
		return Category{}, fmt.Errorf("error updating category: %w", err)
	}

	return c, nil
}

// DeleteCategory deletes a category from the database
func DeleteCategory(db *database.DB, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `DELETE FROM categories WHERE id = $1`

	_, err := db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error deleting category: %w", err)
	}

	return nil
}
