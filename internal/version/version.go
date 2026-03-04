package version

import (
	"fmt"
	"runtime"
)

// Build information. These variables are set via ldflags during build.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// Get returns version info
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns formatted version string
func String() string {
	return fmt.Sprintf("trs version %s\n  Git commit: %s\n  Build date: %s\n  Go version: %s\n  Platform: %s/%s",
		Version,
		GitCommit,
		BuildDate,
		GoVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)
}
