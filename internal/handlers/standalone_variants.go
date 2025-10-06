package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ngenohkevin/kuiper_admin/internal/models"
	"github.com/ngenohkevin/kuiper_admin/internal/templates"
)

// ListProductVariants handles the request to list all product variants
func (h *Handler) ListProductVariants(w http.ResponseWriter, r *http.Request) {
	// Get all variants with retries
	var variants []models.ProductVariant
	var err error

	for retries := 0; retries < 3; retries++ {
		variants, err = models.GetAllProductVariants(h.DB)
		if err == nil {
			break
		}

		log.Printf("Attempt %d: Error getting product variants: %v", retries+1, err)
		time.Sleep(500 * time.Millisecond)
	}

	if err != nil {
		log.Printf("Failed to get product variants after retries: %v", err)
		http.Error(w, fmt.Sprintf("Error getting product variants: %v", err), http.StatusInternalServerError)
		return
	}

	// Get all products for reference with retries
	var products []models.Product

	for retries := 0; retries < 3; retries++ {
		products, err = models.GetAllProducts(h.DB)
		if err == nil {
			break
		}

		log.Printf("Attempt %d: Error getting products: %v", retries+1, err)
		time.Sleep(500 * time.Millisecond)
	}

	if err != nil {
		log.Printf("Failed to get products after retries: %v", err)
		http.Error(w, fmt.Sprintf("Error getting products: %v", err), http.StatusInternalServerError)
		return
	}

	templates.ProductVariantList(variants, products).Render(r.Context(), w)
}

// NewStandaloneVariantForm handles the request to show the form for creating a new product variant
func (h *Handler) NewStandaloneVariantForm(w http.ResponseWriter, r *http.Request) {
	// Get all products for dropdown
	products, err := models.GetAllProducts(h.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting products: %v", err), http.StatusInternalServerError)
		return
	}

	templates.StandaloneProductVariantForm(nil, products, false).Render(r.Context(), w)
}

// EditStandaloneVariantForm handles the request to show the form for editing a product variant
func (h *Handler) EditStandaloneVariantForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing variant ID", http.StatusBadRequest)
		return
	}

	// Get the variant
	variant, err := models.GetProductVariantByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting product variant: %v", err), http.StatusInternalServerError)
		return
	}

	// Get all products for dropdown
	products, err := models.GetAllProducts(h.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting products: %v", err), http.StatusInternalServerError)
		return
	}

	templates.StandaloneProductVariantForm(&variant, products, true).Render(r.Context(), w)
}

// CreateStandaloneVariant handles the request to create a new product variant
func (h *Handler) CreateStandaloneVariant(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	productID := r.FormValue("product_id")
	name := r.FormValue("name")
	priceStr := r.FormValue("price")
	stockCountStr := r.FormValue("stock_count")
	isAvailableStr := r.FormValue("is_available")

	// Validate required fields
	if productID == "" || name == "" || priceStr == "" || stockCountStr == "" {
		http.Error(w, "Product, name, price, and stock count are required", http.StatusBadRequest)
		return
	}

	// Parse numeric values
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		http.Error(w, "Invalid price", http.StatusBadRequest)
		return
	}

	stockCount, err := strconv.Atoi(stockCountStr)
	if err != nil {
		http.Error(w, "Invalid stock count", http.StatusBadRequest)
		return
	}

	// Handle is_available checkbox
	isAvailable := isAvailableStr == "true"

	// Get the product to ensure it exists
	_, err = models.GetProductByID(h.DB, productID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error finding product: %v", err), http.StatusBadRequest)
		return
	}

	// Ensure the product is set to have variants
	err = models.UpdateProductHasVariants(h.DB, productID, true)
	if err != nil {
		log.Printf("Error updating product has_variants flag: %v", err)
		http.Error(w, fmt.Sprintf("Error updating product has_variants flag: %v", err), http.StatusInternalServerError)
		return
	}

	// Create the product variant
	_, err = models.CreateProductVariant(h.DB, productID, name, price, stockCount, isAvailable)
	if err != nil {
		log.Printf("Error creating product variant: %v", err)
		http.Error(w, fmt.Sprintf("Error creating product variant: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the variants list
	http.Redirect(w, r, "/variants", http.StatusSeeOther)
}

// UpdateStandaloneVariant handles the request to update a product variant
func (h *Handler) UpdateStandaloneVariant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing variant ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	productID := r.FormValue("product_id")
	name := r.FormValue("name")
	priceStr := r.FormValue("price")
	stockCountStr := r.FormValue("stock_count")
	isAvailableStr := r.FormValue("is_available")

	// Validate required fields
	if productID == "" || name == "" || priceStr == "" || stockCountStr == "" {
		http.Error(w, "Product, name, price, and stock count are required", http.StatusBadRequest)
		return
	}

	// Get the current variant
	currentVariant, err := models.GetProductVariantByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting product variant: %v", err), http.StatusInternalServerError)
		return
	}

	// If the product ID has changed, we need to handle that
	if currentVariant.ProductID != productID {
		// Make sure the new product exists
		_, err = models.GetProductByID(h.DB, productID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error finding new product: %v", err), http.StatusBadRequest)
			return
		}

		// Ensure the new product is set to have variants
		err = models.UpdateProductHasVariants(h.DB, productID, true)
		if err != nil {
			log.Printf("Error updating new product has_variants flag: %v", err)
			http.Error(w, fmt.Sprintf("Error updating product has_variants flag: %v", err), http.StatusInternalServerError)
			return
		}

		// Check if this was the last variant for the old product
		oldProductVariants, err := models.GetProductVariantsByProductID(h.DB, currentVariant.ProductID)
		if err != nil {
			log.Printf("Error checking variants for old product: %v", err)
		} else if len(oldProductVariants) == 1 && oldProductVariants[0].ID == id {
			// This is the only variant for the old product, so we can set has_variants to false
			err = models.UpdateProductHasVariants(h.DB, currentVariant.ProductID, false)
			if err != nil {
				log.Printf("Error updating old product has_variants flag: %v", err)
			}
		}
	}

	// Parse numeric values
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		http.Error(w, "Invalid price", http.StatusBadRequest)
		return
	}

	stockCount, err := strconv.Atoi(stockCountStr)
	if err != nil {
		http.Error(w, "Invalid stock count", http.StatusBadRequest)
		return
	}

	// Handle is_available checkbox
	isAvailable := isAvailableStr == "true"

	// Update the product variant
	_, err = models.UpdateProductVariantWithProductID(h.DB, id, productID, name, price, stockCount, isAvailable)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating product variant: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the variants list
	http.Redirect(w, r, "/variants", http.StatusSeeOther)
}

// DeleteStandaloneVariant handles the request to delete a product variant
func (h *Handler) DeleteStandaloneVariant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing variant ID", http.StatusBadRequest)
		return
	}

	// Get the variant to find its product ID
	variant, err := models.GetProductVariantByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting product variant: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete the variant
	err = models.DeleteProductVariant(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting product variant: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if this was the last variant for the product
	variants, err := models.GetProductVariantsByProductID(h.DB, variant.ProductID)
	if err != nil {
		log.Printf("Error checking remaining variants for product: %v", err)
	} else if len(variants) == 0 {
		// No more variants for this product, so set has_variants to false
		err = models.UpdateProductHasVariants(h.DB, variant.ProductID, false)
		if err != nil {
			log.Printf("Error updating product has_variants flag: %v", err)
		}
	}

	// For HTMX delete requests, just return 200 OK
	w.WriteHeader(http.StatusOK)
}
