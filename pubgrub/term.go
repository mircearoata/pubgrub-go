package pubgrub

import (
	"fmt"

	"github.com/mircearoata/pubgrub-go/pubgrub/semver"
)

type Term struct {
	pkg               string
	versionConstraint semver.Constraint
	positive          bool
}

func (t Term) Equal(other Term) bool {
	return t.pkg == other.pkg && t.versionConstraint.Equal(other.versionConstraint) && t.positive == other.positive
}

func (t Term) Negate() Term {
	return Term{
		pkg:               t.pkg,
		versionConstraint: t.versionConstraint,
		positive:          !t.positive,
	}
}

// Inverse returns a term that has its positive flag and constraint inverted
// This term will be satisfied by the same versions the original term would,
// but not by a missing term (if the original term was negative)
func (t Term) Inverse() Term {
	return Term{
		pkg:               t.pkg,
		versionConstraint: t.versionConstraint.Inverse(),
		positive:          !t.positive,
	}
}

func (t Term) Dependency() string {
	return t.pkg
}

func (t Term) Constraint() semver.Constraint {
	return t.versionConstraint
}

func (t Term) Positive() bool {
	return t.positive
}

type termRelation int

const (
	termRelationSatisfied termRelation = iota
	termRelationContradicted
	termRelationInconclusive
)

func (t Term) relation(other Term) termRelation {
	if t.pkg != other.pkg {
		return -1
	}

	intersection := t.intersect(other)

	if intersection.Equal(other) {
		return termRelationSatisfied
	}
	if intersection.versionConstraint.IsEmpty() {
		return termRelationContradicted
	}
	return termRelationInconclusive
}

func (t Term) intersect(other Term) Term {
	if t.pkg != other.pkg {
		return Term{}
	}

	switch {
	case t.positive && other.positive:
		return Term{
			pkg:               t.pkg,
			versionConstraint: t.versionConstraint.Intersect(other.versionConstraint),
			positive:          true,
		}
	case t.positive && !other.positive:
		return Term{
			pkg:               t.pkg,
			versionConstraint: t.versionConstraint.Difference(other.versionConstraint),
			positive:          true,
		}
	case !t.positive && other.positive:
		return Term{
			pkg:               t.pkg,
			versionConstraint: other.versionConstraint.Difference(t.versionConstraint),
			positive:          true,
		}
	case !t.positive && !other.positive:
		return Term{
			pkg:               t.pkg,
			versionConstraint: t.versionConstraint.Union(other.versionConstraint),
			positive:          false,
		}
	}

	return Term{}
}

func (t Term) difference(other Term) Term {
	return t.intersect(other.Negate())
}

func (t Term) String() string {
	if t.versionConstraint.IsAny() {
		return fmt.Sprintf("every version of %s", t.pkg)
	}
	return fmt.Sprintf("%s \"%s\"", t.pkg, t.versionConstraint)
}
