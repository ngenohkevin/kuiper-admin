package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ngenohkevin/kuiper_admin/internal/models"
	"github.com/ngenohkevin/kuiper_admin/internal/templates"
)

// CreateProductVariant handles the request to create a new product variant
func (h *Handler) CreateProductVariant(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	if productID == "" {
		http.Error(w, "Missing product ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	priceStr := r.FormValue("price")
	stockCountStr := r.FormValue("stock_count")
	isAvailableStr := r.FormValue("is_available")

	// Validate required fields
	if name == "" || priceStr == "" {
		http.Error(w, "Name and price are required", http.StatusBadRequest)
		return
	}

	// Parse numeric values
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		http.Error(w, "Invalid price", http.StatusBadRequest)
		return
	}

	// If stock count is empty, default to 0
	stockCount := 0
	if stockCountStr != "" {
		stockCount, err = strconv.Atoi(stockCountStr)
		if err != nil {
			http.Error(w, "Invalid stock count", http.StatusBadRequest)
			return
		}
	}

	// Handle is_available checkbox
	isAvailable := isAvailableStr == "true"

	// Create the product variant
	_, err = models.CreateProductVariant(h.DB, productID, name, price, stockCount, isAvailable)
	if err != nil {
		log.Printf("Error creating product variant: %v", err)
		http.Error(w, fmt.Sprintf("Error creating product variant: %v", err), http.StatusInternalServerError)
		return
	}

	// Ensure the product is marked as having variants
	err = models.UpdateProductHasVariants(h.DB, productID, true)
	if err != nil {
		log.Printf("Warning: Error updating product has_variants flag: %v", err)
		// Continue anyway, don't fail the request if this update fails
	}

	// Redirect to the product view
	http.Redirect(w, r, "/products/"+productID, http.StatusSeeOther)
}

// EditProductVariantForm handles the request to show the form for editing a product variant
func (h *Handler) EditProductVariantForm(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	variantID := chi.URLParam(r, "variantID")
	if productID == "" || variantID == "" {
		http.Error(w, "Missing product ID or variant ID", http.StatusBadRequest)
		return
	}

	// Get the parent product
	product, err := models.GetProductByID(h.DB, productID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting product: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the variant
	variant, err := models.GetProductVariantByID(h.DB, variantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting product variant: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if the variant belongs to the specified product
	if variant.ProductID != productID {
		http.Error(w, "Variant does not belong to the specified product", http.StatusBadRequest)
		return
	}

	templates.ProductVariantForm(product, &variant, true).Render(r.Context(), w)
}

// UpdateProductVariant handles the request to update a product variant
func (h *Handler) UpdateProductVariant(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	variantID := chi.URLParam(r, "variantID")
	if productID == "" || variantID == "" {
		http.Error(w, "Missing product ID or variant ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	priceStr := r.FormValue("price")
	stockCountStr := r.FormValue("stock_count")
	isAvailableStr := r.FormValue("is_available")

	// Validate required fields
	if name == "" || priceStr == "" || stockCountStr == "" {
		http.Error(w, "Name, price, and stock count are required", http.StatusBadRequest)
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

	// Update the product variant
	_, err = models.UpdateProductVariant(h.DB, variantID, name, price, stockCount, isAvailable)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating product variant: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the product view
	http.Redirect(w, r, "/products/"+productID, http.StatusSeeOther)
}

// DeleteProductVariant handles the request to delete a product variant
func (h *Handler) DeleteProductVariant(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	variantID := chi.URLParam(r, "variantID")
	if productID == "" || variantID == "" {
		http.Error(w, "Missing product ID or variant ID", http.StatusBadRequest)
		return
	}

	// Delete the product variant
	err := models.DeleteProductVariant(h.DB, variantID)
	if err != nil {
		log.Printf("Error deleting product variant: %v", err)
		http.Error(w, fmt.Sprintf("Error deleting product variant: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully deleted variant %s", variantID)

	// For HTMX delete requests, just return 200 OK
	w.WriteHeader(http.StatusOK)
}
