package version

import (
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

// These constants will be set by ldflags.
// They can be set by goreleaser
// https://goreleaser.com/cookbooks/using-main.version/?h=using+main.version
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

func LogVersion() {
	log := zapr.NewLogger(zap.L())
	log.Info("binary version", "version", Version, "commit", Commit, "date", Date, "builtBy", BuiltBy)
}
