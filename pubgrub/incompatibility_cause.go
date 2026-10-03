package pubgrub

import "github.com/mircearoata/pubgrub-go/pubgrub/semver"

type IncompatibilityCause interface {
	cause()
}

type RootCause struct{}

func (RootCause) cause() {}

type DependencyCause struct {
	Pkg        string
	PkgRange   semver.Constraint
	Target     string
	Constraint semver.Constraint
	Optional   bool
}

func (DependencyCause) cause() {}

type NoVersionsCause struct {
	Pkg        string
	Constraint semver.Constraint
}

func (NoVersionsCause) cause() {}

type ConflictCause struct {
	A *Incompatibility
	B *Incompatibility
}

func (ConflictCause) cause() {}
