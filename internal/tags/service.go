package tags

import "fmt"

type TagService struct {
	repo *TagRepository
}

func NewTagService(repo *TagRepository) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) Create(input *TagCreate) (*Tag, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	return s.repo.Create(input)
}

func (s *TagService) List() ([]Tag, error) {
	return s.repo.List()
}

func (s *TagService) Get(id int64) (*Tag, error) {
	return s.repo.GetByID(id)
}

// AssignToSession assigns tags to a session
func (s *TagService) AssignToSession(sessionID int64, tagIDs []int64) error {
	return s.repo.AssignToSession(sessionID, tagIDs)
}

// RemoveFromSession removes a tag from a session
func (s *TagService) RemoveFromSession(sessionID, tagID int64) error {
	return s.repo.RemoveFromSession(sessionID, tagID)
}

// ListForSession returns all tags for a session
func (s *TagService) ListForSession(sessionID int64) ([]Tag, error) {
	return s.repo.ListForSession(sessionID)
}
