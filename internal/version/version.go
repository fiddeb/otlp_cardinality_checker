// Package version holds build-time version information injected via ldflags.
package version

// Variables are set at build time:
//
//	go build -ldflags "-X github.com/fidde/otlp_cardinality_checker/internal/version.Version=v1.2.3 ..."
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)
