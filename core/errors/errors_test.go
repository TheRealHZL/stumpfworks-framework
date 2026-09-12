package errors

import (
	stderrors "errors"
	"testing"
)

func TestWrapPreservesCauseAndPublicMessage(t *testing.T) {
	cause := stderrors.New("private database detail")
	err := Wrap("database_unavailable", "Service temporarily unavailable", cause)
	if !stderrors.Is(err, cause) {
		t.Fatal("cause was not preserved")
	}
	coded, ok := As(err)
	if !ok || coded.Public != "Service temporarily unavailable" {
		t.Fatalf("unexpected coded error: %#v", coded)
	}
}
