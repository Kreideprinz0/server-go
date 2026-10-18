package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"server-go/storage"
)

// Handler - обёртка над хранилищем
type Handler struct {
	store *storage.Storage
}

func New(store *storage.Storage) *Handler {
	return &Handler{store: store}
}

// Routes регистрирует все маршруты
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /items", h.list)
	mux.HandleFunc("GET /items/{id}", h.get)
	mux.HandleFunc("POST /items", h.create)
	mux.HandleFunc("PUT /items/{id}", h.update)
	mux.HandleFunc("DELETE /items/{id}", h.delete)
}

// вспомогательные функции

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseID(r *http.Request) (int, error) {
	return strconv.Atoi(r.PathValue("id"))
}

// HTTP-обработчики

// GET /items
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.List())
}

// GET /items/{id}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// POST /items
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Price int    `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	item := h.store.Create(req.Name, req.Price)
	writeJSON(w, http.StatusCreated, item)
}

// PUT /items/{id}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Name  string `json:"name"`
		Price int    `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	item, err := h.store.Update(id, req.Name, req.Price)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// DELETE /items/{id}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.store.Delete(id); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
