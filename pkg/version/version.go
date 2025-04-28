package version

// Package version information
var (
	// Version is the current version of hawkeye
	Version = "dev"
	// BuildDate is the date the binary was built
	BuildDate = "unknown"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// UserAgent returns the user agent string
func UserAgent() string {
	return "Hawkeye/" + Version
}
