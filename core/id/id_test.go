package id

import (
	"regexp"
	"testing"
)

func TestNewReturnsUniqueUUIDv4(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	first, err := New()
	if err != nil {
		t.Fatal(err)
	}
	second, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("generated duplicate IDs")
	}
	if !pattern.MatchString(first) {
		t.Fatalf("invalid UUID %q", first)
	}
}
