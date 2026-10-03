package pubgrub

import (
	"errors"

	"github.com/mircearoata/pubgrub-go/pubgrub/semver"
)

type PackageVersion struct {
	Version              semver.Version
	Dependencies         map[string]semver.Constraint
	OptionalDependencies map[string]semver.Constraint
	ForbiddenReason      string // e.g "requires A", "is deprecated"
}

type Source interface {
	GetPackageVersions(pkg string) ([]PackageVersion, error)
	PickVersion(pkg string, version []semver.Version) semver.Version
}

var ErrPackageNotFound = errors.New("package not found")
