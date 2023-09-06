package lib

import (
	"fmt"
	"github.com/mircearoata/pubgrub-go/lib/version"
)

type term struct {
	pkg               string
	versionConstraint version.Constraint
	positive          bool
}

func (t term) Equal(other term) bool {
	return t.pkg == other.pkg && t.versionConstraint.Equal(other.versionConstraint) && t.positive == other.positive
}

func (t term) Negate() term {
	return term{
		pkg:               t.pkg,
		versionConstraint: t.versionConstraint,
		positive:          !t.positive,
	}
}

func (t term) Dependency() string {
	return t.pkg
}

func (t term) Constraint() version.Constraint {
	return t.versionConstraint
}

func (t term) Positive() bool {
	return t.positive
}

func (t term) Satisfies(other term) bool {
	return t.pkg == other.pkg && t.Relation(other) == termRelationSatisfied
}

type termRelation int

const (
	termRelationSatisfied termRelation = iota
	termRelationContradicted
	termRelationInconclusive
)

func (t term) Relation(other term) termRelation {
	if t.pkg != other.pkg {
		return -1
	}

	intersection := t.Intersect(other)

	if intersection.Equal(other) {
		return termRelationSatisfied
	}
	if intersection.versionConstraint.IsEmpty() {
		return termRelationContradicted
	}
	return termRelationInconclusive
}

func (t term) Intersect(other term) term {
	if t.pkg != other.pkg {
		return term{}
	}

	switch {
	case t.positive && other.positive:
		return term{
			pkg:               t.pkg,
			versionConstraint: t.versionConstraint.Intersect(other.versionConstraint),
			positive:          true,
		}
	case t.positive && !other.positive:
		return term{
			pkg:               t.pkg,
			versionConstraint: t.versionConstraint.Difference(other.versionConstraint),
			positive:          true,
		}
	case !t.positive && other.positive:
		return term{
			pkg:               t.pkg,
			versionConstraint: other.versionConstraint.Difference(t.versionConstraint),
			positive:          true,
		}
	case !t.positive && !other.positive:
		return term{
			pkg:               t.pkg,
			versionConstraint: t.versionConstraint.Union(other.versionConstraint),
			positive:          false,
		}
	}

	return term{}
}

func (t term) Difference(other term) term {
	return t.Intersect(other.Negate())
}

func (t term) compatibleDependency(other string) bool {
	return t.pkg == other
}

func (t term) String() string {
	if t.versionConstraint.Inverse().IsEmpty() {
		return fmt.Sprintf("every version of %s", t.pkg)
	}
	return fmt.Sprintf("%s \"%s\"", t.pkg, t.versionConstraint)
}
