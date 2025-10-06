package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/ngenohkevin/kuiper_admin/internal/database"
	"github.com/ngenohkevin/kuiper_admin/internal/models"
	"github.com/ngenohkevin/kuiper_admin/internal/templates"
)

type Handler struct {
	DB      *database.DB
	Session *scs.SessionManager
}

// New creates a new handler instance
func New(db *database.DB, session *scs.SessionManager) *Handler {
	return &Handler{
		DB:      db,
		Session: session,
	}
}

// Home handles the homepage request
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	// Get counts for each entity
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second) // Increased timeout
	defer cancel()

	var categoriesCount, productsCount, reviewsCount int

	// Try to connect to the database and get counts with retries
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		var err1, err2, err3 error

		// Get categories count
		err1 = h.DB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM categories").Scan(&categoriesCount)

		// Get products count
		err2 = h.DB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM products").Scan(&productsCount)

		// Get reviews count
		err3 = h.DB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM reviews").Scan(&reviewsCount)

		// If all queries succeeded, break the loop
		if err1 == nil && err2 == nil && err3 == nil {
			break
		}

		// If this was the last attempt and we still have errors
		if i == maxRetries-1 {
			if err1 != nil {
				log.Printf("Database error getting categories count: %v", err1)
				http.Error(w, "Error getting categories count", http.StatusInternalServerError)
				return
			}
			if err2 != nil {
				log.Printf("Database error getting products count: %v", err2)
				http.Error(w, "Error getting products count", http.StatusInternalServerError)
				return
			}
			if err3 != nil {
				log.Printf("Database error getting reviews count: %v", err3)
				http.Error(w, "Error getting reviews count", http.StatusInternalServerError)
				return
			}
		}

		// Wait a bit before retrying
		time.Sleep(500 * time.Millisecond)
	}

	templates.Home(categoriesCount, productsCount, reviewsCount).Render(r.Context(), w)
}

// CATEGORY HANDLERS

// ListCategories handles the request to list all categories
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	// Check if search query parameter exists
	searchQuery := r.URL.Query().Get("q")

	var categories []models.Category
	var err error

	if searchQuery != "" {
		// If search query exists, search for matching categories
		categories, err = models.SearchCategories(h.DB, searchQuery)
	} else {
		// Otherwise, get all categories
		categories, err = models.GetAllCategories(h.DB)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting categories: %v", err), http.StatusInternalServerError)
		return
	}

	templates.CategoryList(categories).Render(r.Context(), w)
}

// GetCategory handles the request to view a single category
func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing category ID", http.StatusBadRequest)
		return
	}

	category, err := models.GetCategoryByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting category: %v", err), http.StatusInternalServerError)
		return
	}

	// Get all categories for parent lookup
	categories, err := models.GetAllCategories(h.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting categories: %v", err), http.StatusInternalServerError)
		return
	}

	templates.CategoryView(category, categories).Render(r.Context(), w)
}

// NewCategoryForm handles the request to show the form for creating a new category
func (h *Handler) NewCategoryForm(w http.ResponseWriter, r *http.Request) {
	// Get all categories for parent dropdown
	categories, err := models.GetAllCategories(h.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting categories: %v", err), http.StatusInternalServerError)
		return
	}

	templates.CategoryForm(nil, categories, false).Render(r.Context(), w)
}

// EditCategoryForm handles the request to show the form for editing a category
func (h *Handler) EditCategoryForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing category ID", http.StatusBadRequest)
		return
	}

	category, err := models.GetCategoryByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting category: %v", err), http.StatusInternalServerError)
		return
	}

	// Get all categories for parent dropdown
	categories, err := models.GetAllCategories(h.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting categories: %v", err), http.StatusInternalServerError)
		return
	}

	templates.CategoryForm(&category, categories, true).Render(r.Context(), w)
}

// CreateCategory handles the request to create a new category
func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	slug := r.FormValue("slug")
	parentID := r.FormValue("parent_id")

	// Validate required fields
	if name == "" || slug == "" {
		http.Error(w, "Name and slug are required", http.StatusBadRequest)
		return
	}

	// Handle optional parent ID
	var parentIDPtr *string
	if parentID != "" {
		parentIDPtr = &parentID
	}

	// If parent_id is set, verify that it exists
	if parentIDPtr != nil {
		_, err := models.GetCategoryByID(h.DB, *parentIDPtr)
		if err != nil {
			log.Printf("Parent category with ID %s not found: %v", *parentIDPtr, err)
			http.Error(w, fmt.Sprintf("Parent category not found: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Create the category
	_, err := models.CreateCategory(h.DB, name, slug, parentIDPtr)
	if err != nil {
		log.Printf("Error creating category: %v", err)
		http.Error(w, fmt.Sprintf("Error creating category: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the categories list
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

// UpdateCategory handles the request to update a category
func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing category ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	slug := r.FormValue("slug")
	parentID := r.FormValue("parent_id")

	// Validate required fields
	if name == "" || slug == "" {
		http.Error(w, "Name and slug are required", http.StatusBadRequest)
		return
	}

	// Handle optional parent ID
	var parentIDPtr *string
	if parentID != "" {
		parentIDPtr = &parentID
	}

	// Update the category
	_, err := models.UpdateCategory(h.DB, id, name, slug, parentIDPtr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating category: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the category view
	http.Redirect(w, r, "/categories/"+id, http.StatusSeeOther)
}

// DeleteCategory handles the request to delete a category
func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing category ID", http.StatusBadRequest)
		return
	}

	// Delete the category
	err := models.DeleteCategory(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting category: %v", err), http.StatusInternalServerError)
		return
	}

	// For HTMX delete requests, just return 200 OK
	w.WriteHeader(http.StatusOK)
}

// PRODUCT HANDLERS

// ListProducts handles the request to list all products
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	pageSize := 15
	if ps := r.URL.Query().Get("limit"); ps != "" {
		if parsedSize, err := strconv.Atoi(ps); err == nil && parsedSize > 0 && parsedSize <= 100 {
			pageSize = parsedSize
		}
	}

	// Check if search query parameter exists
	searchQuery := r.URL.Query().Get("q")
	categoryID := r.URL.Query().Get("category")

	if searchQuery != "" {
		// If search query exists, search for matching products (no pagination for search yet)
		products, err := models.SearchProducts(h.DB, searchQuery)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error searching products: %v", err), http.StatusInternalServerError)
			return
		}
		templates.ModernProductList(products).Render(r.Context(), w)
	} else {
		// Use pagination
		result, err := models.GetProductsPaginated(h.DB, page, pageSize, categoryID, "")
		if err != nil {
			http.Error(w, fmt.Sprintf("Error getting products: %v", err), http.StatusInternalServerError)
			return
		}

		// Pass pagination result to template with full metadata
		templates.ModernProductListPaginated(*result).Render(r.Context(), w)
	}
}

// GetProduct handles the request to view a single product
func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing product ID", http.StatusBadRequest)
		return
	}

	product, err := models.GetProductByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting product: %v", err), http.StatusInternalServerError)
		return
	}

	templates.ModernProductView(product).Render(r.Context(), w)
}

// NewProductForm handles the request to show the form for creating a new product
func (h *Handler) NewProductForm(w http.ResponseWriter, r *http.Request) {
	// Get all categories for dropdown
	categories, err := models.GetAllCategories(h.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting categories: %v", err), http.StatusInternalServerError)
		return
	}

	templates.ModernProductForm(nil, categories, false).Render(r.Context(), w)
}

// EditProductForm handles the request to show the form for editing a product
func (h *Handler) EditProductForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing product ID", http.StatusBadRequest)
		return
	}

	product, err := models.GetProductByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting product: %v", err), http.StatusInternalServerError)
		return
	}

	// Get all categories for dropdown
	categories, err := models.GetAllCategories(h.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting categories: %v", err), http.StatusInternalServerError)
		return
	}

	// Use only ModernProductForm to fix the duplication issue
	templates.ModernProductForm(&product, categories, true).Render(r.Context(), w)
}

// CreateProduct handles the request to create a new product
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	slug := r.FormValue("slug")
	categoryID := r.FormValue("category_id")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	stockCountStr := r.FormValue("stock_count")
	imageURLsStr := r.FormValue("image_urls")
	isAvailableStr := r.FormValue("is_available")
	enableVariantsStr := r.FormValue("enable_variants")

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

	// Parse image URLs
	var imageURLs []string
	if imageURLsStr != "" {
		// Split by newline and filter empty strings
		for _, url := range strings.Split(imageURLsStr, "\n") {
			trimmedURL := strings.TrimSpace(url)
			if trimmedURL != "" {
				imageURLs = append(imageURLs, trimmedURL)
			}
		}
	}

	// Handle optional category ID
	var categoryIDPtr *string
	if categoryID != "" {
		categoryIDPtr = &categoryID

		// Verify that the category exists
		_, err := models.GetCategoryByID(h.DB, categoryID)
		if err != nil {
			log.Printf("Category with ID %s not found: %v", categoryID, err)
			http.Error(w, fmt.Sprintf("Category not found: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Handle is_available checkbox
	isAvailable := isAvailableStr == "true"

	// Handle has_variants flag
	hasVariants := enableVariantsStr == "true"
	log.Printf("Enable variants: %s, hasVariants: %v", enableVariantsStr, hasVariants)

	// Create the product
	product, err := models.CreateProduct(h.DB, categoryIDPtr, name, slug, description, price, imageURLs, stockCount, isAvailable, hasVariants)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		http.Error(w, fmt.Sprintf("Error creating product: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the product view
	http.Redirect(w, r, "/products/"+product.ID, http.StatusSeeOther)
}

// UpdateProduct handles the request to update a product
func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing product ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	slug := r.FormValue("slug")
	categoryID := r.FormValue("category_id")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	stockCountStr := r.FormValue("stock_count")
	imageURLsStr := r.FormValue("image_urls")
	isAvailableStr := r.FormValue("is_available")
	enableVariantsStr := r.FormValue("enable_variants")

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

	// Parse image URLs
	var imageURLs []string
	if imageURLsStr != "" {
		// Split by newline and filter empty strings
		for _, url := range strings.Split(imageURLsStr, "\n") {
			trimmedURL := strings.TrimSpace(url)
			if trimmedURL != "" {
				imageURLs = append(imageURLs, trimmedURL)
			}
		}
	}

	// Handle optional category ID
	var categoryIDPtr *string
	if categoryID != "" {
		categoryIDPtr = &categoryID
	}

	// Handle is_available checkbox
	isAvailable := isAvailableStr == "true"

	// Handle variants flag
	hasVariants := enableVariantsStr == "true"

	// Get current product to check if it has variants
	currentProduct, err := models.GetProductByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting current product: %v", err), http.StatusInternalServerError)
		return
	}

	// If we're disabling variants but the product has variants, we need to handle this specially
	if !hasVariants && currentProduct.HasVariants && len(currentProduct.Variants) > 0 {
		log.Printf("Warning: Product %s has variants but variants flag is being disabled. Keeping has_variants=true.", id)
		hasVariants = true
	}

	// Update the product first
	_, err = models.UpdateProduct(h.DB, id, categoryIDPtr, name, slug, description, price, imageURLs, stockCount, isAvailable, hasVariants)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating product: %v", err), http.StatusInternalServerError)
		return
	}

	// Handle variants if enabled
	if hasVariants {
		// Process variant data from form (similar to CreateProductWithVariants)
		variantData := make(map[string]map[string]string)

		// Log the form data for debugging
		log.Printf("Processing variants for product update. Form data: %+v", r.Form)

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

		// Create new variants from form data
		successCount := 0
		for idx, data := range variantData {
			variantName := data["name"]
			priceStr := data["price"]
			stockStr := data["stock"]

			// Skip if missing required fields
			if variantName == "" || priceStr == "" || stockStr == "" {
				log.Printf("Skipping variant %s due to missing fields: name=%s, price=%s, stock=%s",
					idx, variantName, priceStr, stockStr)
				continue
			}

			// Parse numeric values
			variantPrice, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				log.Printf("Skipping variant %s due to invalid price: %s", idx, priceStr)
				continue
			}

			variantStockCount, err := strconv.Atoi(stockStr)
			if err != nil {
				log.Printf("Skipping variant %s due to invalid stock: %s", idx, stockStr)
				continue
			}

			// Check if this variant already exists by name (simple check)
			existingVariant := false
			for _, existingVar := range currentProduct.Variants {
				if existingVar.Name == variantName {
					existingVariant = true
					log.Printf("Variant %s already exists, skipping creation", variantName)
					break
				}
			}

			if !existingVariant {
				// Create the new variant
				variant, err := models.CreateProductVariant(h.DB, id, variantName, variantPrice, variantStockCount, true)
				if err != nil {
					log.Printf("Error creating product variant %s: %v", variantName, err)
					continue
				}

				log.Printf("Successfully created variant: %+v", variant)
				successCount++
			}
		}

		log.Printf("Created %d new variants for product %s", successCount, id)
	}

	// Redirect to the product view
	http.Redirect(w, r, "/products/"+id, http.StatusSeeOther)
}

// DeleteProduct handles the request to delete a product
func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing product ID", http.StatusBadRequest)
		return
	}

	// Delete the product
	err := models.DeleteProduct(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting product: %v", err), http.StatusInternalServerError)
		return
	}

	// For HTMX delete requests, just return 200 OK
	w.WriteHeader(http.StatusOK)
}

// REVIEW HANDLERS

// ListReviews handles the request to list all reviews
func (h *Handler) ListReviews(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	pageSize := 15
	if ps := r.URL.Query().Get("limit"); ps != "" {
		if parsedSize, err := strconv.Atoi(ps); err == nil && parsedSize > 0 && parsedSize <= 100 {
			pageSize = parsedSize
		}
	}

	// Check if search query parameter exists
	searchQuery := r.URL.Query().Get("q")

	if searchQuery != "" {
		// If search query exists, search for matching reviews (no pagination for search yet)
		reviews, err := models.SearchReviews(h.DB, searchQuery)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error searching reviews: %v", err), http.StatusInternalServerError)
			return
		}
		templates.ReviewList(reviews).Render(r.Context(), w)
	} else {
		// Use pagination
		result, err := models.GetReviewsPaginated(h.DB, page, pageSize)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error getting reviews: %v", err), http.StatusInternalServerError)
			return
		}

		// Pass pagination result to template - using existing template with just data for now
		templates.ReviewList(result.Data).Render(r.Context(), w)
	}
}

// GetReview handles the request to view a single review
func (h *Handler) GetReview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing review ID", http.StatusBadRequest)
		return
	}

	review, err := models.GetReviewByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting review: %v", err), http.StatusInternalServerError)
		return
	}

	err = templates.ReviewView(review).Render(r.Context(), w)
	if err != nil {
		return
	}
}

// NewReviewForm handles the request to show the form for creating a new review
func (h *Handler) NewReviewForm(w http.ResponseWriter, r *http.Request) {
	// Get all products for dropdown
	products, err := models.GetAllProducts(h.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting products: %v", err), http.StatusInternalServerError)
		return
	}

	err = templates.ReviewForm(nil, products, false).Render(r.Context(), w)
	if err != nil {
		return
	}
}

// EditReviewForm handles the request to show the form for editing a review
func (h *Handler) EditReviewForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing review ID", http.StatusBadRequest)
		return
	}

	review, err := models.GetReviewByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting review: %v", err), http.StatusInternalServerError)
		return
	}

	// Get all products for dropdown
	products, err := models.GetAllProducts(h.DB)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting products: %v", err), http.StatusInternalServerError)
		return
	}

	templates.ReviewForm(&review, products, true).Render(r.Context(), w)
}

// CreateReview handles the request to create a new review
func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	productID := r.FormValue("product_id")
	ratingStr := r.FormValue("rating")
	comment := r.FormValue("comment")
	reviewerName := r.FormValue("reviewer_name")

	// Validate required fields
	if productID == "" || ratingStr == "" {
		http.Error(w, "Product and rating are required", http.StatusBadRequest)
		return
	}

	// Verify that the product exists
	_, productErr := models.GetProductByID(h.DB, productID)
	if productErr != nil {
		log.Printf("Product with ID %s not found: %v", productID, productErr)
		http.Error(w, fmt.Sprintf("Product not found: %v", productErr), http.StatusBadRequest)
		return
	}

	// Parse rating
	rating, err := strconv.ParseFloat(ratingStr, 64)
	if err != nil || rating < 1 || rating > 5 {
		http.Error(w, "Invalid rating", http.StatusBadRequest)
		return
	}

	// Create the review with an empty session ID for now
	var sessionIDPtr *string
	_, err = models.CreateReview(h.DB, &productID, sessionIDPtr, rating, comment, reviewerName)
	if err != nil {
		log.Printf("Error creating review: %v", err)
		http.Error(w, fmt.Sprintf("Error creating review: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the reviews list
	http.Redirect(w, r, "/reviews", http.StatusSeeOther)
}

// UpdateReview handles the request to update a review
func (h *Handler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing review ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	productID := r.FormValue("product_id")
	ratingStr := r.FormValue("rating")
	comment := r.FormValue("comment")
	reviewerName := r.FormValue("reviewer_name")

	// Validate required fields
	if productID == "" || ratingStr == "" {
		http.Error(w, "Product and rating are required", http.StatusBadRequest)
		return
	}

	// Parse rating
	rating, err := strconv.ParseFloat(ratingStr, 64)
	if err != nil || rating < 1 || rating > 5 {
		http.Error(w, "Invalid rating", http.StatusBadRequest)
		return
	}

	// Update the review with an empty session ID for now
	var sessionIDPtr *string
	_, err = models.UpdateReview(h.DB, id, &productID, sessionIDPtr, rating, comment, reviewerName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating review: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the review view
	http.Redirect(w, r, "/reviews/"+id, http.StatusSeeOther)
}

// DeleteReview handles the request to delete a review
func (h *Handler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing review ID", http.StatusBadRequest)
		return
	}

	// Delete the review
	err := models.DeleteReview(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting review: %v", err), http.StatusInternalServerError)
		return
	}

	// For HTMX delete requests, just return 200 OK
	w.WriteHeader(http.StatusOK)
}

// AUTH HANDLERS

// LoginPage displays the login form
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	// If user is already authenticated, redirect to home
	if h.Session.GetBool(r.Context(), "authenticated") {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get any error message from the query string
	errorMsg := r.URL.Query().Get("error")

	err := templates.Login(errorMsg).Render(r.Context(), w)
	if err != nil {
		return
	}
}

// Login handles the login form submission
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Get username and password from form
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Check credentials - hardcoded for simplicity
	if username == "dylstar" && password == "dylstarperi@4560" {
		// Set user as authenticated
		h.Session.Put(r.Context(), "authenticated", true)
		h.Session.Put(r.Context(), "username", username)

		// Redirect to home page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Invalid credentials
	http.Redirect(w, r, "/login?error=Invalid+username+or+password", http.StatusSeeOther)
}

// Logout handles user logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Destroy the session
	err := h.Session.Destroy(r.Context())
	if err != nil {
		return
	}

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ImageProxy handles proxying external images to avoid CORS issues
func (h *Handler) ImageProxy(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		http.Error(w, "Missing URL parameter", http.StatusBadRequest)
		return
	}

	// Create HTTP client with timeout and headers
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request with headers
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// Add headers to avoid blocking and handle authentication
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GanymedeAdmin/1.0)")
	req.Header.Set("Accept", "image/*,*/*")
	req.Header.Set("Referer", "https://pixshelf.perigrine.cloud")
	req.Header.Set("Cache-Control", "no-cache")

	// Fetch the image
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Image proxy error for %s: %v", imageURL, err)
		http.Error(w, "Failed to fetch image", http.StatusBadGateway)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		log.Printf("Image proxy got status %d for %s", resp.StatusCode, imageURL)
		http.Error(w, fmt.Sprintf("Image not found (status: %d)", resp.StatusCode), http.StatusNotFound)
		return
	}

	// Set appropriate headers
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "image/jpeg") // default
	}

	// Set caching and CORS headers
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Copy the image data
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying image data: %v", err)
	}
}
