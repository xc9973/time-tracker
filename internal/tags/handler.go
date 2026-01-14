package tags

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"time-tracker/internal/shared/errors"
)

type TagsHandler struct {
	service *TagService
}

func NewTagsHandler(svc *TagService) *TagsHandler {
	return &TagsHandler{service: svc}
}

func (h *TagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/api/v1/tags" && r.Method == http.MethodPost:
		h.Create(w, r)
	case path == "/api/v1/tags" && r.Method == http.MethodGet:
		h.List(w, r)
	case strings.HasPrefix(path, "/api/v1/tags/") && r.Method == http.MethodGet:
		h.Get(w, r)
	default:
		errors.WriteError(w, errors.NotFoundError("Endpoint not found"))
	}
}

func (h *TagsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input TagCreate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errors.WriteError(w, errors.ValidationError("Invalid JSON body"))
		return
	}
	created, err := h.service.Create(&input)
	if err != nil {
		if strings.Contains(err.Error(), "validation error") {
			errors.WriteError(w, errors.ValidationError(strings.TrimPrefix(err.Error(), "validation error: ")))
			return
		}
		errors.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *TagsHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List()
	if err != nil {
		errors.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (h *TagsHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tags/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		errors.WriteError(w, errors.ValidationError("Invalid id"))
		return
	}
	tag, err := h.service.Get(id)
	if err != nil {
		errors.WriteError(w, err)
		return
	}
	if tag == nil {
		errors.WriteError(w, errors.NotFoundError("Tag not found"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tag)
}
