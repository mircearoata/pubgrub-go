package pubgrub

import "github.com/mircearoata/pubgrub-go/pubgrub/semver"

type PackageVersion struct {
	Version              semver.Version
	Dependencies         map[string]semver.Constraint
	OptionalDependencies map[string]semver.Constraint
}

type Source interface {
	GetPackageVersions(pkg string) ([]PackageVersion, error)
	PickVersion(pkg string, version []semver.Version) semver.Version
}
