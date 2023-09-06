package lib

import "github.com/mircearoata/pubgrub-go/lib/version"

type Source interface {
	GetPackageVersions(pkg string) ([]version.Version, error)
	GetPackageVersionsSatisfying(pkg string, constraint version.Constraint) ([]version.Version, error)
	GetPackageVersionDependencies(pkg string, version version.Version) (map[string]version.Constraint, error)
}
