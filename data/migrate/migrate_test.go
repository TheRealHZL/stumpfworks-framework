package migrate

import (
	"testing"
	"testing/fstest"
)

func TestLoadSortsAndChecksumsMigrations(t *testing.T) {
	source := fstest.MapFS{"sql/002_second.up.sql": {Data: []byte("SELECT 2")}, "sql/001_first.up.sql": {Data: []byte("SELECT 1")}, "sql/README.md": {Data: []byte("ignored")}}
	migrations, err := Load(source, "sql")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(migrations) != 2 || migrations[0].Version != 1 || migrations[1].Version != 2 {
		t.Fatalf("unexpected order: %#v", migrations)
	}
	if migrations[0].Checksum == "" {
		t.Fatal("checksum is empty")
	}
}

func TestLoadRejectsDuplicateVersions(t *testing.T) {
	source := fstest.MapFS{"sql/001_first.up.sql": {Data: []byte("SELECT 1")}, "sql/001_other.up.sql": {Data: []byte("SELECT 2")}}
	if _, err := Load(source, "sql"); err == nil {
		t.Fatal("expected duplicate version error")
	}
}
