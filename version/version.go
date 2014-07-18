package version

import (
	"github.com/coreos/go-semver/semver"
)

const Version = "0.0.4"

var SemVersion semver.Version

func init() {
	sv, err := semver.NewVersion(Version)
	if err != nil {
		panic("bad version string!")
	}
	SemVersion = *sv
}
