package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ngenohkevin/kuiper_admin/internal/models"
	"github.com/ngenohkevin/kuiper_admin/internal/templates"
)

// ListSessions handles the request to list all sessions
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	// Check if search query parameter exists
	searchQuery := r.URL.Query().Get("q")

	var sessions []models.Session
	var err error

	if searchQuery != "" {
		// If search query exists, search for matching sessions
		sessions, err = models.SearchSessions(h.DB, searchQuery)
	} else {
		// Otherwise, get all sessions
		sessions, err = models.GetAllSessions(h.DB)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting sessions: %v", err), http.StatusInternalServerError)
		return
	}

	templates.SessionList(sessions).Render(r.Context(), w)
}

// GetSession handles the request to view a single session
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	session, err := models.GetSessionByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting session: %v", err), http.StatusInternalServerError)
		return
	}

	templates.SessionView(session).Render(r.Context(), w)
}

// EditSessionForm handles the request to show the form for editing a session
func (h *Handler) EditSessionForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	session, err := models.GetSessionByID(h.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting session: %v", err), http.StatusInternalServerError)
		return
	}

	templates.SessionForm(session).Render(r.Context(), w)
}

// UpdateSession handles the request to update a session
func (h *Handler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	token := r.FormValue("token")
	dataStr := r.FormValue("data")
	expiresAtStr := r.FormValue("expires_at")

	// Validate required fields
	if token == "" || expiresAtStr == "" {
		http.Error(w, "Token and expires_at are required", http.StatusBadRequest)
		return
	}

	// Parse session data JSON
	var data json.RawMessage
	if dataStr != "" {
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON data: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		data = json.RawMessage("{}")
	}

	// Parse expires_at datetime
	expiresAt, err := time.Parse("2006-01-02T15:04", expiresAtStr)
	if err != nil {
		http.Error(w, "Invalid expires_at date format", http.StatusBadRequest)
		return
	}

	// Update the session
	_, err = models.UpdateSession(h.DB, id, token, data, expiresAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating session: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the session view
	http.Redirect(w, r, "/sessions/"+id, http.StatusSeeOther)
}

// DeleteSession handles the request to delete a session
func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// Delete the session
	err := models.DeleteSession(h.DB, id)
	if err != nil {
		log.Printf("Error deleting session: %v", err)
		http.Error(w, fmt.Sprintf("Error deleting session: %v", err), http.StatusInternalServerError)
		return
	}

	// For HTMX delete requests - always return a redirect to the sessions page
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/sessions")
		w.WriteHeader(http.StatusOK)
		return
	}

	// For regular requests, redirect to the sessions list
	http.Redirect(w, r, "/sessions", http.StatusSeeOther)
}
