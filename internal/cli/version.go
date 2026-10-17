package cli

import (
	"runtime/debug"
	"strings"
)

// version is set at build time with
// -ldflags "-X github.com/dorkitude/deadlore/internal/cli.version=X.Y.Z"
// (GoReleaser does). `go install ...@vX.Y.Z` builds fall back to the module
// version; anything else reports "dev".
var version = "dev"

func currentVersion() string {
	if version != "" && version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}
