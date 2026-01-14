package tags

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"time-tracker/internal/shared/errors"
)

// SessionTagsRequest is the request body for assigning tags to a session
type SessionTagsRequest struct {
	TagIDs []int64 `json:"tag_ids"`
}

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
	// Session-tags association endpoints
	case strings.HasPrefix(path, "/api/v1/sessions/") && strings.HasSuffix(path, "/tags"):
		switch r.Method {
		case http.MethodPost:
			h.AssignTagsToSession(w, r)
		case http.MethodGet:
			h.ListSessionTags(w, r)
		default:
			errors.WriteError(w, errors.NotFoundError("Method not allowed"))
		}
	case strings.HasPrefix(path, "/api/v1/sessions/") && strings.Count(path, "/") == 6:
		// DELETE /api/v1/sessions/:id/tags/:tag_id
		if r.Method == http.MethodDelete {
			h.RemoveTagFromSession(w, r)
		} else {
			errors.WriteError(w, errors.NotFoundError("Method not allowed"))
		}
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

// AssignTagsToSession assigns tags to a session
func (h *TagsHandler) AssignTagsToSession(w http.ResponseWriter, r *http.Request) {
	// Extract session ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	path = strings.TrimSuffix(path, "/tags")
	sessionID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || sessionID <= 0 {
		errors.WriteError(w, errors.ValidationError("Invalid session id"))
		return
	}

	var input SessionTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errors.WriteError(w, errors.ValidationError("Invalid JSON body"))
		return
	}

	if err := h.service.AssignToSession(sessionID, input.TagIDs); err != nil {
		errors.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveTagFromSession removes a tag from a session
func (h *TagsHandler) RemoveTagFromSession(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		errors.WriteError(w, errors.ValidationError("Invalid path"))
		return
	}

	sessionID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || sessionID <= 0 {
		errors.WriteError(w, errors.ValidationError("Invalid session id"))
		return
	}

	tagID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || tagID <= 0 {
		errors.WriteError(w, errors.ValidationError("Invalid tag id"))
		return
	}

	if err := h.service.RemoveFromSession(sessionID, tagID); err != nil {
		errors.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListSessionTags lists all tags for a session
func (h *TagsHandler) ListSessionTags(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	path = strings.TrimSuffix(path, "/tags")
	sessionID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || sessionID <= 0 {
		errors.WriteError(w, errors.ValidationError("Invalid session id"))
		return
	}

	tags, err := h.service.ListForSession(sessionID)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tags)
}
