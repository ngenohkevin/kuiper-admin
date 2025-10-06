package middleware

import (
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
)

// Auth creates an authentication middleware with the given session manager
func Auth(sessionManager *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exclude login page, static files, and image proxy from auth check
			if r.URL.Path == "/login" || strings.HasPrefix(r.URL.Path, "/static/") || r.URL.Path == "/proxy/image" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if user is authenticated using the session manager
			if !sessionManager.GetBool(r.Context(), "authenticated") {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
