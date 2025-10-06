package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ngenohkevin/kuiper_admin/internal/models"
	"github.com/ngenohkevin/kuiper_admin/internal/templates"
)

// EnhancedProductForm displays the form for creating a new product with optional variants
func (h *Handler) EnhancedProductForm(w http.ResponseWriter, r *http.Request) {
	// Get all categories for dropdown
	categories, err := models.GetAllCategories(h.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting categories: %v", err), http.StatusInternalServerError)
		return
	}

	templates.EnhancedProductForm(nil, categories, false).Render(r.Context(), w)
}

// CreateProductWithVariants handles the request to create a new product with optional variants
func (h *Handler) CreateProductWithVariants(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Extract product data
	name := r.FormValue("name")
	slug := r.FormValue("slug")
	categoryID := r.FormValue("category_id")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	stockCountStr := r.FormValue("stock_count")
	isAvailableStr := r.FormValue("is_available")
	enableVariantsStr := r.FormValue("enable_variants")
	imageURLsRaw := r.FormValue("image_urls")

	// Validate required fields
	if name == "" || slug == "" || priceStr == "" || stockCountStr == "" {
		http.Error(w, "Name, slug, price, and stock count are required", http.StatusBadRequest)
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

	// Parse image URLs
	var imageURLs []string
	if imageURLsRaw != "" {
		for _, url := range strings.Split(imageURLsRaw, "\n") {
			url = strings.TrimSpace(url)
			if url != "" {
				imageURLs = append(imageURLs, url)
			}
		}
	}

	// Determine if variants are enabled
	enableVariants := enableVariantsStr == "true"

	// Initialize category ID pointer
	var categoryIDPtr *string
	if categoryID != "" {
		categoryIDPtr = &categoryID
	}

	// Create the product
	product, err := models.CreateProduct(
		h.DB,
		categoryIDPtr,
		name,
		slug,
		description,
		price,
		imageURLs,
		stockCount,
		isAvailable,
		enableVariants,
	)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		http.Error(w, fmt.Sprintf("Error creating product: %v", err), http.StatusInternalServerError)
		return
	}

	// If variants are enabled, create the variants
	if enableVariants {
		// Extract variants from form
		variantData := make(map[string]map[string]string)

		// Log the form data for debugging
		log.Printf("Form data: %+v", r.Form)

		for key, values := range r.Form {
			if !strings.HasPrefix(key, "variants[") || len(values) == 0 {
				continue
			}

			// Extract index and field from format "variants[0][name]"
			parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(key, "variants["), "]"), "][")
			if len(parts) != 2 {
				log.Printf("Invalid variant key format: %s", key)
				continue
			}

			index, field := parts[0], parts[1]

			if _, exists := variantData[index]; !exists {
				variantData[index] = make(map[string]string)
			}

			variantData[index][field] = values[0]
			log.Printf("Found variant data: index=%s, field=%s, value=%s", index, field, values[0])
		}

		// Log the extracted variant data
		log.Printf("Extracted variant data: %+v", variantData)

		// Create all variants
		successCount := 0
		for idx, data := range variantData {
			name := data["name"]
			priceStr := data["price"]
			stockStr := data["stock"]

			// Skip if missing required fields
			if name == "" || priceStr == "" || stockStr == "" {
				log.Printf("Skipping variant %s due to missing fields: name=%s, price=%s, stock=%s",
					idx, name, priceStr, stockStr)
				continue
			}

			// Parse numeric values
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				log.Printf("Skipping variant %s due to invalid price: %s", idx, priceStr)
				continue
			}

			stockCount, err := strconv.Atoi(stockStr)
			if err != nil {
				log.Printf("Skipping variant %s due to invalid stock: %s", idx, stockStr)
				continue
			}

			// Create the variant
			variant, err := models.CreateProductVariant(h.DB, product.ID, name, price, stockCount, true)
			if err != nil {
				log.Printf("Error creating product variant %s: %v", name, err)
				continue
			}

			log.Printf("Successfully created variant: %+v", variant)
			successCount++
		}

		log.Printf("Created %d variants for product %s", successCount, product.ID)
	}

	// Redirect to the product view
	http.Redirect(w, r, "/products/"+product.ID, http.StatusSeeOther)
}
