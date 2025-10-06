package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ngenohkevin/kuiper_admin/internal/models"
	"github.com/ngenohkevin/kuiper_admin/internal/templates"
)

// CreateBulkVariants handles the request to create multiple variants at once
func (h *Handler) CreateBulkVariants(w http.ResponseWriter, r *http.Request) {
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

	weightsStr := r.FormValue("weights")

	// Get the parent product
	product, err := models.GetProductByID(h.DB, productID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting product: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse weights from the comma-separated string
	weights := strings.Split(weightsStr, ",")
	if len(weights) == 0 {
		http.Error(w, "No weights provided", http.StatusBadRequest)
		return
	}

	// Create variants for each weight
	for _, weight := range weights {
		weight = strings.TrimSpace(weight)
		if weight == "" {
			continue
		}

		// Add "g" suffix if not present and not a template with custom naming
		name := weight
		if !strings.HasSuffix(strings.ToLower(weight), "g") &&
			!strings.HasSuffix(strings.ToLower(weight), "gram") &&
			!strings.HasSuffix(strings.ToLower(weight), "grams") {
			name = weight + "g"
		}

		// Calculate price based on base product price and weight
		weightValue, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(weight, "g"), "gram"), "grams"), 64)
		if err != nil {
			// Use the product price as fallback
			weightValue = 1.0
		}

		// Simple price calculation, adjust as needed
		price := product.Price * (1 + (weightValue / 100))

		// Round price to 2 decimal places
		price = float64(int(price*100)) / 100

		// Create the variant
		_, err = models.CreateProductVariant(h.DB, productID, name, price, 0, true)
		if err != nil {
			log.Printf("Error creating variant %s: %v", name, err)
		}
	}

	// Ensure the product is marked as having variants
	err = models.UpdateProductHasVariants(h.DB, productID, true)
	if err != nil {
		log.Printf("Warning: Error updating product has_variants flag: %v", err)
	}

	// Redirect to the product view
	http.Redirect(w, r, "/products/"+productID, http.StatusSeeOther)
}

// GetVariantEditForm handles the request for the variant edit form via API
func (h *Handler) GetVariantEditForm(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	variantID := chi.URLParam(r, "variantID")

	if productID == "" || variantID == "" {
		log.Printf("Missing parameter - productID: %s, variantID: %s", productID, variantID)
		http.Error(w, "Missing product ID or variant ID", http.StatusBadRequest)
		return
	}

	// Get the parent product
	product, err := models.GetProductByID(h.DB, productID)
	if err != nil {
		log.Printf("Error getting product %s: %v", productID, err)
		http.Error(w, fmt.Sprintf("Error getting product: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the variant
	variant, err := models.GetProductVariantByID(h.DB, variantID)
	if err != nil {
		log.Printf("Error getting variant %s: %v", variantID, err)
		http.Error(w, fmt.Sprintf("Error getting product variant: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if the variant belongs to the specified product
	if variant.ProductID != productID {
		log.Printf("Variant %s does not belong to product %s (belongs to %s)",
			variantID, productID, variant.ProductID)
		http.Error(w, "Variant does not belong to the specified product", http.StatusBadRequest)
		return
	}

	log.Printf("Rendering edit form for variant %s (name: %s) of product %s",
		variant.ID, variant.Name, product.ID)
	templates.VariantEditForm(product, variant).Render(r.Context(), w)
}

// UpdateVariantAPI handles the API request to update a product variant
func (h *Handler) UpdateVariantAPI(w http.ResponseWriter, r *http.Request) {
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

	// Get updated product for rendering updated variants
	product, err := models.GetProductByID(h.DB, productID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting updated product: %v", err), http.StatusInternalServerError)
		return
	}

	// Render just the variants table rows for HTMX swap
	w.Header().Set("Content-Type", "text/html")
	for _, variant := range product.Variants {
		err := templates.VariantRow(variant, productID).Render(r.Context(), w)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error rendering variant row: %v", err), http.StatusInternalServerError)
			return
		}
	}
}
