// Package version exposes build metadata embedded by an application build.
package version

import "runtime/debug"

// These values may be set with -ldflags -X during release builds.
var (
	Version   = "dev"
	Commit    string
	BuildTime string
)

// Info describes a running build.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	Dirty     bool   `json:"dirty,omitempty"`
}

// Current returns build metadata, falling back to development values.
func Current() Info {
	info := Info{Version: Version, Commit: Commit, BuildTime: BuildTime}
	build, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}
	if info.Version == "dev" && build.Main.Version != "" && build.Main.Version != "(devel)" {
		info.Version = build.Main.Version
	}
	for _, setting := range build.Settings {
		switch setting.Key {
		case "vcs.revision":
			if info.Commit == "" {
				info.Commit = setting.Value
			}
		case "vcs.time":
			if info.BuildTime == "" {
				info.BuildTime = setting.Value
			}
		case "vcs.modified":
			info.Dirty = setting.Value == "true"
		}
	}
	return info
}
