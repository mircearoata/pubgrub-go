package lib

import "github.com/mircearoata/pubgrub-go/lib/version"
import "slices"

type assignment interface {
	Package() string
	DecisionLevel() int
}

type partialSolution struct {
	assignments []assignment
}

type derivation struct {
	t             term
	cause         *Incompatibility
	decisionLevel int
}

func (d derivation) Package() string {
	return d.t.pkg
}

func (d derivation) DecisionLevel() int {
	return d.decisionLevel
}

type decision struct {
	pkg           string
	version       version.Version
	decisionLevel int
}

func (d decision) Package() string {
	return d.pkg
}

func (d decision) DecisionLevel() int {
	return d.decisionLevel
}

func (ps *partialSolution) get(pkg string) *term {
	var result *term
	for _, a := range ps.assignments {
		if a.Package() == pkg {
			if dec, ok := a.(decision); ok {
				return &term{
					pkg:               dec.pkg,
					versionConstraint: version.SingleVersionConstraint(dec.version),
					positive:          true,
				}
			}
			if der, ok := a.(derivation); ok {
				if result == nil {
					result = &der.t
				} else {
					intersection := result.Intersect(der.t)
					result = &intersection
				}
			}
		}
	}
	return result
}

func (ps *partialSolution) currentDecisionLevel() int {
	currentDecisionLevel := 0
	for _, a := range ps.assignments {
		if _, ok := a.(decision); ok {
			currentDecisionLevel++
		}
	}
	return currentDecisionLevel
}

func (ps *partialSolution) add(t term, cause *Incompatibility) {
	newDerivation := derivation{
		t:             t,
		decisionLevel: ps.currentDecisionLevel(),
		cause:         cause,
	}

	ps.assignments = append(ps.assignments, newDerivation)
}

func (ps *partialSolution) prefix(size int) partialSolution {
	return partialSolution{
		assignments: slices.Clone(ps.assignments[:size]),
	}
}

func (ps *partialSolution) findPositiveUndecided() string {
	var decidedPackages []string
	for _, a := range ps.assignments {
		if _, ok := a.(decision); ok {
			decidedPackages = append(decidedPackages, a.Package())
		}
	}
	for _, a := range ps.assignments {
		if der, ok := a.(derivation); ok {
			if der.t.positive && !slices.Contains(decidedPackages, der.t.pkg) {
				return der.t.pkg
			}
		}
	}
	return ""
}

func (ps *partialSolution) decisionsMap() map[string]version.Version {
	result := map[string]version.Version{}
	for _, a := range ps.assignments {
		if dec, ok := a.(decision); ok {
			result[dec.pkg] = dec.version
		}
	}
	return result
}
