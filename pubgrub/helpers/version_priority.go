package helpers

import (
	"github.com/mircearoata/pubgrub-go/pubgrub/semver"
)

// StandardVersionPriority returns the latest release version if there is one, otherwise the latest prerelease version.
// NOTE: versions must be sorted in increasing order and must not be empty.
func StandardVersionPriority(versions []semver.Version) semver.Version {
	var latestRelease, latestPrerelease *semver.Version
	for _, v := range versions {
		v := v
		if v.IsPrerelease() {
			if latestPrerelease == nil || latestPrerelease.Compare(v) < 0 {
				latestPrerelease = &v
			}
		} else {
			if latestRelease == nil || latestRelease.Compare(v) < 0 {
				latestRelease = &v
			}
		}
	}
	if latestRelease != nil {
		return *latestRelease
	}
	return *latestPrerelease
}
