package templates

import "strings"

// Helper functions for templates

// GetImageSrc validates image URLs and returns appropriate src for external URLs
func GetImageSrc(url string) string {
	if url == "" {
		return "/static/img/placeholder.svg"
	}

	// If it's a relative URL or already using proxy, use as-is
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "/proxy/image") {
		return url
	}

	// For external URLs, use direct first (proxy will be fallback on error)
	return url
}

// getTitle returns the appropriate title based on whether we're editing or creating
func getTitle(isEdit bool) string {
	if isEdit {
		return "Edit Category"
	}
	return "New Category"
}

// getProductTitle returns the appropriate title for product forms
func getProductTitle(isEdit bool) string {
	if isEdit {
		return "Edit Product"
	}
	return "New Product"
}

// getReviewTitle returns the appropriate title for review forms
func getReviewTitle(isEdit bool) string {
	if isEdit {
		return "Edit Review"
	}
	return "New Review"
}
