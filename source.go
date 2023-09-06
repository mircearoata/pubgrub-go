package pubgrub

import (
	"github.com/mircearoata/pubgrub-go/version"
)

type PackageVersion struct {
	Version              version.Version
	Dependencies         map[string]version.Constraint
	OptionalDependencies map[string]version.Constraint
}

type Source interface {
	GetPackageVersions(pkg string) ([]PackageVersion, error)
	PickVersion(pkg string, version []version.Version) version.Version
}
