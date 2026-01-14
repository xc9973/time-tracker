package tags

import (
	"errors"
	"strings"

	"time-tracker/internal/shared/validation"
)

type Tag struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	CreatedAt string `json:"created_at"`
}

type TagCreate struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

var ErrNameRequired = errors.New("name is required")

func (t *TagCreate) Validate() error {
	t.Name = validation.SanitizeString(t.Name)
	t.Color = strings.TrimSpace(t.Color)

	if t.Name == "" {
		return ErrNameRequired
	}

	if t.Color == "" {
		t.Color = "#6B7280"
	}

	return nil
}
