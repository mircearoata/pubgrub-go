package helpers

import (
	"slices"

	"github.com/mircearoata/pubgrub-go/pubgrub/semver"
)

// StandardVersionPriority returns the latest release version if there is one, otherwise the latest prerelease version.
// NOTE: versions must be sorted in increasing order and must not be empty.
func StandardVersionPriority(versions []semver.Version) semver.Version {
	if len(versions) == 0 {
		return semver.Version{}
	}
	for _, version := range slices.Backward(versions) {
		if !version.IsPrerelease() {
			return version
		}
	}
	return versions[len(versions)-1]
}
