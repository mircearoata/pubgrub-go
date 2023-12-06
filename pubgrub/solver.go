package pubgrub

import (
	"maps"
	"slices"

	"github.com/mircearoata/pubgrub-go/pubgrub/semver"
	"github.com/mircearoata/pubgrub-go/pubgrub/util"
	"github.com/pkg/errors"
)

type solver struct {
	rootPkg           string
	incompatibilities []*Incompatibility
	partialSolution   partialSolution

	source Source
}

func Solve(source Source, rootPkg string) (map[string]semver.Version, error) {
	s := solver{
		source:  source,
		rootPkg: rootPkg,
		incompatibilities: []*Incompatibility{
			{
				terms: map[string]Term{
					rootPkg: {
						pkg:               rootPkg,
						versionConstraint: semver.AnyConstraint,
						positive:          false,
					},
				},
			},
		},
	}

	next := rootPkg

	for {
		err := s.unitPropagation(next)
		if err != nil {
			return nil, err
		}

		// Prefetch all positive undecided packages
		undecided := s.partialSolution.allPositiveUndecided()
		go func() {
			for _, pkg := range undecided {
				go func(pkg string) {
					_, _ = s.source.GetPackageVersions(pkg)
				}(pkg)
			}
		}()

		var done bool
		next, done, err = s.decision()
		if err != nil {
			return nil, errors.Wrap(err, "failed to make decision")
		}
		if done {
			break
		}
	}

	result := s.partialSolution.decisionsMap()
	delete(result, rootPkg)
	return result, nil
}

func (s *solver) unitPropagation(inPkg string) error {
	changed := []string{inPkg}
	var contradictedIncompatibilities []*Incompatibility
	for len(changed) > 0 {
		pkg := changed[0]
		changed = changed[1:]

		for i := len(s.incompatibilities) - 1; i >= 0; i-- {
			currentIncompatibility := s.incompatibilities[i]
			if slices.Contains(contradictedIncompatibilities, currentIncompatibility) {
				continue
			}
			hasPkg := false
			for _, t := range currentIncompatibility.terms {
				if t.pkg == pkg {
					hasPkg = true
					break
				}
			}
			if !hasPkg {
				continue
			}

			rel, t := currentIncompatibility.relation(&s.partialSolution)
			if rel == setRelationSatisfied {
				newIncompatibility, err := s.conflictResolution(currentIncompatibility)
				if err != nil {
					return err
				}
				newRel, newT := newIncompatibility.relation(&s.partialSolution)
				if newRel != setRelationAlmostSatisfied {
					return errors.New("new incompatibility is not almost satisfied, this should never happen")
				}
				s.partialSolution.add(newT.Negate(), newIncompatibility)
				changed = []string{newT.pkg}
				contradictedIncompatibilities = append(contradictedIncompatibilities, newIncompatibility)
				break
			} else if rel == setRelationAlmostSatisfied {
				s.partialSolution.add(t.Negate(), currentIncompatibility)
				changed = append(changed, t.pkg)
			}
			contradictedIncompatibilities = append(contradictedIncompatibilities, currentIncompatibility)
		}
	}
	return nil
}

func (s *solver) conflictResolution(fromIncompatibility *Incompatibility) (*Incompatibility, error) {
	incompatibilityChanged := false
	for {
		if s.isIncompatibilityTerminal(fromIncompatibility) {
			return nil, SolvingError{fromIncompatibility}
		}

		satisfierIdx := util.BinarySearchFunc(0, len(s.partialSolution.assignments), func(i int) bool {
			prefix := s.partialSolution.prefix(i + 1)
			rel, _ := fromIncompatibility.relation(&prefix)
			return rel == setRelationSatisfied
		})
		satisfier := s.partialSolution.assignments[satisfierIdx]

		incompatibilityTerm := fromIncompatibility.get(satisfier.Package())

		previousSatisfierIdx := util.BinarySearchFunc(-1, satisfierIdx+1, func(i int) bool {
			prefix := s.partialSolution.prefix(i + 1)
			prefix.assignments = append(prefix.assignments, satisfier)
			rel, _ := fromIncompatibility.relation(&prefix)
			return rel == setRelationSatisfied
		})
		var previousSatisfier assignment
		previousSatisfierLevel := 1
		if previousSatisfierIdx >= 0 {
			previousSatisfier = s.partialSolution.assignments[previousSatisfierIdx]
			previousSatisfierLevel = previousSatisfier.DecisionLevel()
		}

		if _, ok := satisfier.(decision); ok || previousSatisfierLevel != satisfier.DecisionLevel() {
			if incompatibilityChanged {
				s.addIncompatibility(fromIncompatibility)
			}

			decLevel := 0
			for i := 0; i < len(s.partialSolution.assignments); i++ {
				if _, ok := s.partialSolution.assignments[i].(decision); ok {
					decLevel++
					if decLevel > previousSatisfierLevel {
						s.partialSolution = s.partialSolution.prefix(i)
						break
					}
				}
			}

			return fromIncompatibility, nil
		}

		der := satisfier.(derivation)

		priorCause := fromIncompatibility.makePriorCause(der.cause, satisfier.Package())

		if rel := incompatibilityTerm.relation(der.t); rel != termRelationSatisfied {
			priorCause.add(der.t.difference(*incompatibilityTerm).Negate())
		}

		fromIncompatibility = priorCause
		incompatibilityChanged = true
	}
}

func (s *solver) decision() (string, bool, error) {
	pkg := s.partialSolution.findPositiveUndecided()
	if pkg == "" {
		return "", true, nil
	}

	t := s.partialSolution.get(pkg)

	versions, err := s.source.GetPackageVersions(t.pkg)
	if err != nil {
		return pkg, false, errors.Wrap(err, "failed to get package versions")
	}

	availableVersions := make([]semver.Version, 0, len(versions))
	for _, v := range versions {
		availableVersions = append(availableVersions, v.Version)
	}

	// Sort versions in ascending order
	slices.SortFunc(availableVersions, func(a, b semver.Version) int {
		return a.Compare(b)
	})

	var compatibleVersions []semver.Version
	for _, v := range availableVersions {
		if t.versionConstraint.Contains(v) {
			compatibleVersions = append(compatibleVersions, v)
		}
	}

	if len(versions) == 0 || len(compatibleVersions) == 0 {
		s.addIncompatibility(&Incompatibility{
			terms: map[string]Term{pkg: *t},
		})
		return pkg, false, nil
	}

	chosenVersion := s.source.PickVersion(t.pkg, compatibleVersions)

	if !slices.ContainsFunc(compatibleVersions, func(v semver.Version) bool {
		return v.Compare(chosenVersion) == 0
	}) {
		return pkg, false, errors.New("chosen version not compatible")
	}

	var chosenVersionData *PackageVersion
	for _, v := range versions {
		if v.Version.Compare(chosenVersion) == 0 {
			chosenVersionData = &v
			break
		}
	}

	// Add dependencies in a deterministic order (alphabetical)
	deps := make([]string, 0, len(chosenVersionData.Dependencies))
	for dep := range chosenVersionData.Dependencies {
		deps = append(deps, dep)
	}
	slices.Sort(deps)
	for _, dep := range deps {
		constraint := chosenVersionData.Dependencies[dep]
		var versionsWithThisDependency []semver.Version
		for _, v := range versions {
			if vDep, ok := v.Dependencies[dep]; ok && constraint.Equal(vDep) {
				versionsWithThisDependency = append(versionsWithThisDependency, v.Version)
			}
		}
		slices.SortFunc(versionsWithThisDependency, func(a, b semver.Version) int {
			return a.Compare(b)
		})
		s.addIncompatibility(&Incompatibility{
			terms: map[string]Term{
				pkg: {
					pkg:               pkg,
					versionConstraint: semver.NewConstraintFromVersionSubset(versionsWithThisDependency, availableVersions),
					positive:          true,
				},
				dep: {
					pkg:               dep,
					versionConstraint: constraint,
				},
			},
			dependant: pkg,
		})
	}

	// Add optional dependencies in a deterministic order (alphabetical)
	optionalDeps := make([]string, 0, len(chosenVersionData.OptionalDependencies))
	for dep := range chosenVersionData.OptionalDependencies {
		optionalDeps = append(optionalDeps, dep)
	}
	slices.Sort(optionalDeps)
	for _, dep := range optionalDeps {
		constraint := chosenVersionData.OptionalDependencies[dep]
		var versionsWithThisDependency []semver.Version
		for _, v := range versions {
			if vDep, ok := v.OptionalDependencies[dep]; ok && constraint.Equal(vDep) {
				versionsWithThisDependency = append(versionsWithThisDependency, v.Version)
			}
		}
		slices.SortFunc(versionsWithThisDependency, func(a, b semver.Version) int {
			return a.Compare(b)
		})
		s.addIncompatibility(&Incompatibility{
			terms: map[string]Term{
				pkg: {
					pkg:               pkg,
					versionConstraint: semver.NewConstraintFromVersionSubset(versionsWithThisDependency, availableVersions),
					positive:          true,
				},
				dep: {
					pkg: dep,
					// A negative term is satisfied if the dependency exists with an incompatible version,
					// or if the dependency does not exist at all.
					// So we use a positive term with an inverse constraint instead,
					// which is satisfied when the dependency exists with an incompatible version
					versionConstraint: constraint.Inverse(),
					positive:          true,
				},
			},
			dependant: pkg,
		})
	}

	s.partialSolution.assignments = append(s.partialSolution.assignments, decision{
		pkg:           t.pkg,
		version:       chosenVersion,
		decisionLevel: s.partialSolution.currentDecisionLevel() + 1,
	})

	return pkg, false, nil
}

func (s *solver) addIncompatibility(in *Incompatibility) {
	if slices.ContainsFunc(s.incompatibilities, func(i *Incompatibility) bool {
		return maps.EqualFunc(i.terms, in.terms, func(a, b Term) bool {
			return a.Equal(b)
		})
	}) {
		return
	}
	s.incompatibilities = append(s.incompatibilities, in)
}

func (s *solver) isIncompatibilityTerminal(in *Incompatibility) bool {
	if len(in.terms) == 0 {
		return true
	}
	if len(in.terms) == 1 {
		for _, t := range in.terms {
			if t.positive && t.pkg == s.rootPkg {
				return true
			}
		}
	}
	return false
}
