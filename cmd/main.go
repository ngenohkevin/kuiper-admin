package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/ngenohkevin/kuiper_admin/internal/database"
	"github.com/ngenohkevin/kuiper_admin/internal/handlers"
	custommiddleware "github.com/ngenohkevin/kuiper_admin/internal/middleware"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize the database connection
	db, err := database.New()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize session manager
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour // Set session lifetime
	sessionManager.Cookie.Persist = true     // Persist cookie after browser close
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Secure = false // Set to true in production with HTTPS

	// Set up router and middleware
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// Custom method override middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				if method := r.PostFormValue("_method"); method != "" {
					r.Method = method
				}
			}
			next.ServeHTTP(w, r)
		})
	})
	r.Use(sessionManager.LoadAndSave)
	r.Use(custommiddleware.Auth(sessionManager))

	// Serve static files
	fs := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static", fs))

	// Initialize handlers with database connection and session manager
	h := handlers.New(db, sessionManager)

	// Image proxy for external images (before auth middleware)
	r.Get("/proxy/image", h.ImageProxy)

	// Define routes
	r.Route("/", func(r chi.Router) {
		// Auth routes
		r.Get("/login", h.LoginPage)
		r.Post("/login", h.Login)
		r.Get("/logout", h.Logout)

		// Main app routes
		r.Get("/", h.Home)

		// Categories routes
		r.Route("/categories", func(r chi.Router) {
			r.Get("/", h.ListCategories)
			r.Get("/new", h.NewCategoryForm)
			r.Post("/", h.CreateCategory)
			r.Get("/{id}", h.GetCategory)
			r.Get("/{id}/edit", h.EditCategoryForm)
			r.Put("/{id}", h.UpdateCategory)
			r.Delete("/{id}", h.DeleteCategory)
		})

		// Products routes
		r.Route("/products", func(r chi.Router) {
			r.Get("/", h.ListProducts)
			r.Get("/new", h.NewProductForm)
			r.Post("/", h.CreateProduct)
			r.Get("/{id}", h.GetProduct)
			r.Get("/{id}/edit", h.EditProductForm)
			r.Put("/{id}", h.UpdateProduct)
			r.Delete("/{id}", h.DeleteProduct)

			// Product variants routes
			r.Post("/{id}/bulk-variants", h.CreateBulkVariants)
			r.Post("/{id}/variants", h.CreateProductVariant)
			r.Get("/{id}/variants/{variantID}/edit", h.EditProductVariantForm)
			r.Put("/{id}/variants/{variantID}", h.UpdateProductVariant)
			r.Delete("/{id}/variants/{variantID}", h.DeleteProductVariant)
		})

		// API Routes for variants - these need to be at the top level
		r.Route("/api/v1/products", func(r chi.Router) {
			r.Get("/{id}/variants/{variantID}/edit-form", h.GetVariantEditForm)
			r.Put("/{id}/variants/{variantID}", h.UpdateVariantAPI)
		})

		// Reviews routes
		r.Route("/reviews", func(r chi.Router) {
			r.Get("/", h.ListReviews)
			r.Get("/new", h.NewReviewForm)
			r.Post("/", h.CreateReview)
			r.Get("/{id}", h.GetReview)
			r.Get("/{id}/edit", h.EditReviewForm)
			r.Put("/{id}", h.UpdateReview)
			r.Delete("/{id}", h.DeleteReview)
		})

		// Sessions routes
		r.Route("/sessions", func(r chi.Router) {
			r.Get("/", h.ListSessions)
			r.Get("/{id}", h.GetSession)
			r.Get("/{id}/edit", h.EditSessionForm)
			r.Put("/{id}", h.UpdateSession)
			r.Delete("/{id}", h.DeleteSession)
		})
	})

	// Create HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		fmt.Printf("Server starting on port %s...\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stopChan

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}

	fmt.Println("Server gracefully stopped")
}
