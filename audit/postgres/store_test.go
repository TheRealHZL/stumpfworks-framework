package postgres

import "testing"

func TestNewRejectsNilPool(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("accepted nil pool")
	}
}
