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

type Review struct {
	ID           string           `json:"id"`
	ProductID    *string          `json:"product_id"`
	SessionID    *string          `json:"session_id"`
	Rating       float64          `json:"rating"`
	Comment      string           `json:"comment"`
	CreatedAt    pgtype.Timestamp `json:"created_at"`
	ReviewerName *string          `json:"reviewer_name"`
	Product      *Product         `json:"product,omitempty"`
}

// GetAllReviews retrieves all reviews from the database
func GetAllReviews(db *database.DB) ([]Review, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT r.id, r.product_id, r.session_id, r.rating, r.comment, r.created_at, r.reviewer_name,
		       p.id, p.name, p.slug
		FROM reviews r
		LEFT JOIN products p ON r.product_id = p.id
		ORDER BY r.created_at DESC
	`

	log.Printf("Executing SQL query: %s", query)
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		log.Printf("Database error: %v", err)
		return nil, fmt.Errorf("error querying reviews: %w", err)
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
			log.Printf("Scan error: %v", err)
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

// GetReviewsPaginated retrieves reviews with pagination
func GetReviewsPaginated(db *database.DB, page, pageSize int) (PaginatedResult[Review], error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate page and pageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Get total count
	countQuery := "SELECT COUNT(*) FROM reviews"
	var totalCount int64
	err := db.Pool.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return PaginatedResult[Review]{}, fmt.Errorf("error counting reviews: %w", err)
	}

	// Get paginated reviews
	query := `
		SELECT r.id, r.product_id, r.session_id, r.rating, r.comment, r.created_at, r.reviewer_name,
		       p.id, p.name, p.slug
		FROM reviews r
		LEFT JOIN products p ON r.product_id = p.id
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`

	log.Printf("Executing paginated SQL query: %s with LIMIT %d OFFSET %d", query, pageSize, offset)
	rows, err := db.Pool.Query(ctx, query, pageSize, offset)
	if err != nil {
		log.Printf("Database error: %v", err)
		return PaginatedResult[Review]{}, fmt.Errorf("error querying reviews: %w", err)
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
			log.Printf("Scan error: %v", err)
			return PaginatedResult[Review]{}, fmt.Errorf("error scanning review row: %w", err)
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
		return PaginatedResult[Review]{}, fmt.Errorf("error iterating review rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	hasNext := page < totalPages
	hasPrev := page > 1

	return PaginatedResult[Review]{
		Data:       reviews,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}, nil
}

// GetReviewByID retrieves a single review by ID
func GetReviewByID(db *database.DB, id string) (Review, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT r.id, r.product_id, r.session_id, r.rating, r.comment, r.created_at, r.reviewer_name,
		       p.id, p.name, p.slug
		FROM reviews r
		LEFT JOIN products p ON r.product_id = p.id
		WHERE r.id = $1
	`

	log.Printf("Executing SQL query for review ID %s: %s", id, query)

	var r Review
	var productID, productName, productSlug string

	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&r.ID, &r.ProductID, &r.SessionID, &r.Rating, &r.Comment, &r.CreatedAt, &r.ReviewerName,
		&productID, &productName, &productSlug,
	)
	if err != nil {
		log.Printf("Database error finding review %s: %v", id, err)
		return Review{}, fmt.Errorf("error finding review: %w", err)
	}

	if productID != "" {
		r.Product = &Product{
			ID:   productID,
			Name: productName,
			Slug: productSlug,
		}
	}

	return r, nil
}

// CreateReview creates a new review in the database
func CreateReview(db *database.DB, productID *string, sessionID *string, rating float64, comment string, reviewerName string) (Review, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Generate a UUID for the new review
	newID := uuid.New().String()

	// Handle optional reviewer_name
	var reviewerNamePtr *string
	if reviewerName != "" {
		reviewerNamePtr = &reviewerName
	}

	// Log the input parameters for debugging
	prodIDValue := "<nil>"
	if productID != nil {
		prodIDValue = *productID
	}

	sessIDValue := "<nil>"
	if sessionID != nil {
		sessIDValue = *sessionID
	}

	reviewerNameValue := "<nil>"
	if reviewerNamePtr != nil {
		reviewerNameValue = *reviewerNamePtr
	}

	log.Printf("Creating review with id=%s, product_id=%s, session_id=%s, rating=%.1f, reviewer_name=%s",
		newID, prodIDValue, sessIDValue, rating, reviewerNameValue)

	query := `
		INSERT INTO reviews (id, product_id, session_id, rating, comment, reviewer_name, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		RETURNING id, product_id, session_id, rating, comment, created_at, reviewer_name
	`

	var r Review
	err := db.Pool.QueryRow(ctx, query, newID, productID, sessionID, rating, comment, reviewerNamePtr).Scan(
		&r.ID, &r.ProductID, &r.SessionID, &r.Rating, &r.Comment, &r.CreatedAt, &r.ReviewerName,
	)
	if err != nil {
		log.Printf("Database error creating review: %v", err)
		return Review{}, fmt.Errorf("error creating review: %w", err)
	}

	log.Printf("Successfully created review with ID: %s", r.ID)
	return r, nil
}

// UpdateReview updates an existing review in the database
func UpdateReview(db *database.DB, id string, productID *string, sessionID *string, rating float64, comment string, reviewerName string) (Review, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Handle optional reviewer_name
	var reviewerNamePtr *string
	if reviewerName != "" {
		reviewerNamePtr = &reviewerName
	}

	query := `
		UPDATE reviews
		SET product_id = $2, session_id = $3, rating = $4, comment = $5, reviewer_name = $6
		WHERE id = $1
		RETURNING id, product_id, session_id, rating, comment, created_at, reviewer_name
	`

	var r Review
	err := db.Pool.QueryRow(ctx, query, id, productID, sessionID, rating, comment, reviewerNamePtr).Scan(
		&r.ID, &r.ProductID, &r.SessionID, &r.Rating, &r.Comment, &r.CreatedAt, &r.ReviewerName,
	)
	if err != nil {
		return Review{}, fmt.Errorf("error updating review: %w", err)
	}

	return r, nil
}

// DeleteReview deletes a review from the database
func DeleteReview(db *database.DB, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `DELETE FROM reviews WHERE id = $1`

	_, err := db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error deleting review: %w", err)
	}

	return nil
}
