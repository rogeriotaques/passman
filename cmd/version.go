package cmd

import (
	"fmt"
	"regexp"
	"strings"
)

// version is the fallback used when no version is embedded (tests, tools).
var version = "dev"

// semverPattern matches MAJOR.MINOR.PATCH with optional pre-release and build
// metadata. Leading zeros are rejected, as the semver spec requires.
var semverPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

// SetVersion validates v as semver and makes it the version reported by
// --version. The build embeds VERSION.
func SetVersion(v string) error {
	v = strings.TrimSpace(v)
	if err := validateVersion(v); err != nil {
		return err
	}
	version = v
	rootCmd.Version = v
	rootCmd.Long = bannerText()
	return nil
}

func validateVersion(v string) error {
	if v == "" {
		return fmt.Errorf("version is empty")
	}
	if !semverPattern.MatchString(v) {
		return fmt.Errorf("version %q is not valid semver (expected MAJOR.MINOR.PATCH)", v)
	}
	return nil
}
