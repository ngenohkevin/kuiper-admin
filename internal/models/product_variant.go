package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/ngenohkevin/kuiper_admin/internal/database"
)

type ProductVariant struct {
	ID          string   `json:"id"`
	ProductID   string   `json:"product_id,omitempty"` // Used for UI display, not in JSONB
	Name        string   `json:"name,omitempty"`
	Price       float64  `json:"price"`
	StockCount  int      `json:"stock_count"`
	IsAvailable bool     `json:"is_available"`
	Weight      string   `json:"weight,omitempty"` // New field for weight/quantity
	Product     *Product `json:"product,omitempty"`
}

// GetAllProductVariants retrieves all product variants from the database
func GetAllProductVariants(db *database.DB) ([]ProductVariant, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Query all products that have variants
	query := `
		SELECT id, variants
		FROM products
		WHERE has_variants = true AND variants IS NOT NULL AND variants != '[]'::jsonb
		ORDER BY name
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying products with variants: %w", err)
	}
	defer rows.Close()

	var allVariants []ProductVariant
	for rows.Next() {
		var productID string
		var variantsJSON []byte

		if err := rows.Scan(&productID, &variantsJSON); err != nil {
			return nil, fmt.Errorf("error scanning product row: %w", err)
		}

		// Parse variants from JSONB
		if variantsJSON != nil && string(variantsJSON) != "[]" && string(variantsJSON) != "null" {
			var variants []ProductVariant
			if err := json.Unmarshal(variantsJSON, &variants); err != nil {
				log.Printf("Error parsing variants JSON: %v", err)
				continue
			}

			// Add product ID to each variant and append to all variants
			for i := range variants {
				variants[i].ProductID = productID
				// If weight field is populated, use it as the name
				if variants[i].Weight != "" && variants[i].Name == "" {
					variants[i].Name = variants[i].Weight
				}
				allVariants = append(allVariants, variants[i])
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return allVariants, nil
}

// GetProductVariantsByProductID retrieves all variants for a product
func GetProductVariantsByProductID(db *database.DB, productID string) ([]ProductVariant, error) {
	// Use the GetProductByID function to get the product with variants
	product, err := GetProductByID(db, productID)
	if err != nil {
		return nil, fmt.Errorf("error getting product: %w", err)
	}

	// Return the variants from the product
	return product.Variants, nil
}

// GetProductVariantByID retrieves a single product variant by ID
func GetProductVariantByID(db *database.DB, id string) (ProductVariant, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	log.Printf("Looking for variant with ID: %s", id)

	// Try a different approach for Supabase - using a JSON object for comparison
	jsonPattern := fmt.Sprintf(`[{"id":"%s"}]`, id)
	rawQuery := `
		SELECT id, variants
		FROM products
		WHERE has_variants = true 
		  AND variants @> $1
	`

	log.Printf("Executing query with jsonPattern: %s", jsonPattern)

	var productID string
	var variantsJSON []byte

	err := db.Pool.QueryRow(ctx, rawQuery, jsonPattern).Scan(&productID, &variantsJSON)
	if err != nil {
		log.Printf("Error finding product with variant %s: %v", id, err)
		return ProductVariant{}, fmt.Errorf("error finding product with variant: %w", err)
	}

	log.Printf("Found variant in product %s", productID)

	// Parse variants from JSONB
	if variantsJSON != nil && string(variantsJSON) != "[]" && string(variantsJSON) != "null" {
		var variants []ProductVariant
		if err := json.Unmarshal(variantsJSON, &variants); err != nil {
			log.Printf("Error parsing variants JSON: %v", err)
			return ProductVariant{}, fmt.Errorf("error parsing variants JSON: %w", err)
		}

		// Find the variant with the matching ID
		for _, v := range variants {
			if v.ID == id {
				v.ProductID = productID
				// If weight field is populated, use it as the name
				if v.Weight != "" && v.Name == "" {
					v.Name = v.Weight
				}
				log.Printf("Found variant: %s, name: %s, price: %.2f", v.ID, v.Name, v.Price)
				return v, nil
			}
		}
	}

	log.Printf("Variant %s not found in product %s", id, productID)
	return ProductVariant{}, fmt.Errorf("variant not found")
}

// CreateProductVariant creates a new product variant in the database
func CreateProductVariant(db *database.DB, productID, name string,
	price float64, stockCount int, isAvailable bool) (ProductVariant, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Generate a UUID for the new product variant
	newID := uuid.New().String()

	// First, get the current product and its variants
	var variantsJSON []byte
	err := db.Pool.QueryRow(ctx, "SELECT variants FROM products WHERE id = $1", productID).Scan(&variantsJSON)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error finding product: %w", err)
	}

	// Parse the existing variants
	var variants []ProductVariant
	if variantsJSON != nil && string(variantsJSON) != "null" {
		if err := json.Unmarshal(variantsJSON, &variants); err != nil {
			return ProductVariant{}, fmt.Errorf("error parsing variants JSON: %w", err)
		}
	}

	// Create the new variant
	newVariant := ProductVariant{
		ID:          newID,
		Name:        name,
		Weight:      name, // Use name as weight since this is what you mentioned you need
		Price:       price,
		StockCount:  stockCount,
		IsAvailable: isAvailable,
	}

	// Add the new variant to the array
	variants = append(variants, newVariant)

	// Convert the variants array back to JSON
	updatedVariantsJSON, err := json.Marshal(variants)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error marshaling variants to JSON: %w", err)
	}

	// Update the product with the new variants array and ensure has_variants is true
	_, err = db.Pool.Exec(ctx,
		"UPDATE products SET variants = $1, has_variants = true, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		updatedVariantsJSON, productID)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error updating product variants: %w", err)
	}

	// Set the ProductID for the return value (it's not stored in the JSON)
	newVariant.ProductID = productID

	return newVariant, nil
}

// UpdateProductVariant updates an existing product variant in the database
func UpdateProductVariant(db *database.DB, id, name string,
	price float64, stockCount int, isAvailable bool) (ProductVariant, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Try a different approach for Supabase - using a JSON object for comparison
	jsonPattern := fmt.Sprintf(`[{"id":"%s"}]`, id)
	rawQuery := `
		SELECT id, variants
		FROM products
		WHERE has_variants = true 
		  AND variants @> $1
	`

	log.Printf("Update variant - executing query with jsonPattern: %s", jsonPattern)

	var productID string
	var variantsJSON []byte

	err := db.Pool.QueryRow(ctx, rawQuery, jsonPattern).Scan(&productID, &variantsJSON)
	if err != nil {
		log.Printf("Error finding product with variant: %v", err)
		return ProductVariant{}, fmt.Errorf("error finding product with variant: %w", err)
	}

	log.Printf("Update variant - found in product %s", productID)

	// Parse the variants
	var variants []ProductVariant
	if err := json.Unmarshal(variantsJSON, &variants); err != nil {
		return ProductVariant{}, fmt.Errorf("error parsing variants JSON: %w", err)
	}

	// Find and update the variant
	var updatedVariant ProductVariant
	for i, v := range variants {
		if v.ID == id {
			// Update the variant
			variants[i].Name = name
			variants[i].Weight = name // Use name as weight
			variants[i].Price = price
			variants[i].StockCount = stockCount
			variants[i].IsAvailable = isAvailable

			// Save for return
			updatedVariant = variants[i]
			updatedVariant.ProductID = productID
			break
		}
	}

	if updatedVariant.ID == "" {
		return ProductVariant{}, fmt.Errorf("variant not found")
	}

	// Convert the updated variants array back to JSON
	updatedVariantsJSON, err := json.Marshal(variants)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error marshaling variants to JSON: %w", err)
	}

	// Update the product with the modified variants array
	_, err = db.Pool.Exec(ctx,
		"UPDATE products SET variants = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		updatedVariantsJSON, productID)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error updating product variants: %w", err)
	}

	return updatedVariant, nil
}

// DeleteProductVariant deletes a product variant from the database
func DeleteProductVariant(db *database.DB, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Try a different approach for Supabase - using a JSON object for comparison
	jsonPattern := fmt.Sprintf(`[{"id":"%s"}]`, id)
	rawQuery := `
		SELECT id, variants
		FROM products
		WHERE has_variants = true 
		  AND variants @> $1
	`

	log.Printf("Delete variant - executing query with jsonPattern: %s", jsonPattern)

	var productID string
	var variantsJSON []byte

	err := db.Pool.QueryRow(ctx, rawQuery, jsonPattern).Scan(&productID, &variantsJSON)
	if err != nil {
		log.Printf("Error finding product with variant: %v", err)
		return fmt.Errorf("error finding product with variant: %w", err)
	}

	log.Printf("Delete variant - found in product %s, JSON: %s", productID, string(variantsJSON))

	// Parse the variants
	var variants []ProductVariant
	if err := json.Unmarshal(variantsJSON, &variants); err != nil {
		log.Printf("Error parsing variants JSON: %v", err)
		return fmt.Errorf("error parsing variants JSON: %w", err)
	}

	// Filter out the variant to delete
	var newVariants []ProductVariant
	for _, v := range variants {
		if v.ID != id {
			newVariants = append(newVariants, v)
		}
	}

	log.Printf("Delete variant - filtered variants from %d to %d", len(variants), len(newVariants))

	// Convert the filtered variants array back to JSON
	newVariantsJSON, err := json.Marshal(newVariants)
	if err != nil {
		log.Printf("Error marshaling variants to JSON: %v", err)
		return fmt.Errorf("error marshaling variants to JSON: %w", err)
	}

	// Update the product with the new variants array
	// Also update has_variants flag if there are no more variants
	hasVariants := len(newVariants) > 0
	_, err = db.Pool.Exec(ctx,
		"UPDATE products SET variants = $1, has_variants = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3",
		newVariantsJSON, hasVariants, productID)
	if err != nil {
		log.Printf("Error updating product variants: %v", err)
		return fmt.Errorf("error updating product variants: %w", err)
	}

	return nil
}

// DeleteProductVariantsByProductID deletes all variants for a product
func DeleteProductVariantsByProductID(db *database.DB, productID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Update the product to have an empty variants array and set has_variants to false
	_, err := db.Pool.Exec(ctx,
		"UPDATE products SET variants = '[]'::jsonb, has_variants = false, updated_at = CURRENT_TIMESTAMP WHERE id = $1",
		productID)
	if err != nil {
		return fmt.Errorf("error clearing product variants: %w", err)
	}

	return nil
}

// UpdateProductVariantWithProductID updates an existing product variant in the database including product ID
// This is more complex as it involves moving the variant from one product to another
func UpdateProductVariantWithProductID(db *database.DB, id, newProductID, name string,
	price float64, stockCount int, isAvailable bool) (ProductVariant, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// First, find the current product that contains this variant
	jsonPattern := fmt.Sprintf(`[{"id":"%s"}]`, id)
	findQuery := `
		SELECT id, variants
		FROM products
		WHERE has_variants = true 
		  AND variants @> $1
	`

	log.Printf("UpdateProductVariantWithProductID - executing query with jsonPattern: %s", jsonPattern)

	var currentProductID string
	var currentVariantsJSON []byte

	err := db.Pool.QueryRow(ctx, findQuery, jsonPattern).Scan(&currentProductID, &currentVariantsJSON)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error finding product with variant: %w", err)
	}

	// Parse the current variants
	var currentVariants []ProductVariant
	if err := json.Unmarshal(currentVariantsJSON, &currentVariants); err != nil {
		return ProductVariant{}, fmt.Errorf("error parsing current variants JSON: %w", err)
	}

	// Find the variant to move
	var variantToMove ProductVariant
	var remainingVariants []ProductVariant

	for _, v := range currentVariants {
		if v.ID == id {
			variantToMove = v
			// Update the variant data
			variantToMove.Name = name
			variantToMove.Weight = name // Use name as weight
			variantToMove.Price = price
			variantToMove.StockCount = stockCount
			variantToMove.IsAvailable = isAvailable
		} else {
			remainingVariants = append(remainingVariants, v)
		}
	}

	if variantToMove.ID == "" {
		return ProductVariant{}, fmt.Errorf("variant not found")
	}

	// Begin a transaction
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error starting transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Step 1: Update the current product's variants array
	currentVariantsJSON, err = json.Marshal(remainingVariants)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error marshaling current variants to JSON: %w", err)
	}

	hasVariants := len(remainingVariants) > 0
	_, err = tx.Exec(ctx,
		"UPDATE products SET variants = $1, has_variants = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3",
		currentVariantsJSON, hasVariants, currentProductID)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error updating current product variants: %w", err)
	}

	// Step 2: Get the new product's variants
	var newVariantsJSON []byte
	err = tx.QueryRow(ctx, "SELECT variants FROM products WHERE id = $1", newProductID).Scan(&newVariantsJSON)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error finding new product: %w", err)
	}

	// Parse the new product's variants
	var newVariants []ProductVariant
	if newVariantsJSON != nil && string(newVariantsJSON) != "null" {
		if err := json.Unmarshal(newVariantsJSON, &newVariants); err != nil {
			return ProductVariant{}, fmt.Errorf("error parsing new variants JSON: %w", err)
		}
	}

	// Add the variant to the new product's variants
	newVariants = append(newVariants, variantToMove)
	updatedNewVariantsJSON, err := json.Marshal(newVariants)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error marshaling new variants to JSON: %w", err)
	}

	// Step 3: Update the new product's variants
	_, err = tx.Exec(ctx,
		"UPDATE products SET variants = $1, has_variants = true, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		updatedNewVariantsJSON, newProductID)
	if err != nil {
		return ProductVariant{}, fmt.Errorf("error updating new product variants: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return ProductVariant{}, fmt.Errorf("error committing transaction: %w", err)
	}

	// Set the ProductID for the return value
	variantToMove.ProductID = newProductID

	return variantToMove, nil
}
