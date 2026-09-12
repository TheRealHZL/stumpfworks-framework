package version

import "testing"

func TestCurrentHonorsLinkedValues(t *testing.T) {
	previousVersion, previousCommit, previousBuildTime := Version, Commit, BuildTime
	t.Cleanup(func() { Version, Commit, BuildTime = previousVersion, previousCommit, previousBuildTime })
	Version, Commit, BuildTime = "0.1.0-test", "abc123", "2026-09-12T00:00:00Z"
	info := Current()
	if info.Version != Version || info.Commit != Commit || info.BuildTime != BuildTime {
		t.Fatalf("unexpected info: %#v", info)
	}
}
