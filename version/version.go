package version

// Version value is provided at compile time by -ldflags
var version string

// Version returns a version string or n/a if not available.
func Version() string {
	if version == "" {
		return "n/a"
	}
	return version
}
