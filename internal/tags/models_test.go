package tags

import "testing"

func TestTag_Validate_NameRequired(t *testing.T) {
	tag := TagCreate{Name: "   "}
	if err := tag.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}
