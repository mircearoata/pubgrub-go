package pubgrub

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/mircearoata/pubgrub-go/pubgrub/semver"
	"github.com/mircearoata/pubgrub-go/pubgrub/util"
)

type solver struct {
	rootPkg           string
	incompatibilities []*Incompatibility
	partialSolution   partialSolution

	source Source
}

type SolveOption func(*solverConfig)

type solverConfig struct {
	environmentPackages map[string]semver.Constraint
}

func WithEnvironmentPackages(packages map[string]semver.Constraint) func(*solverConfig) {
	return func(o *solverConfig) {
		o.environmentPackages = packages
	}
}

func Solve(source Source, rootPkg string, opts ...SolveOption) (map[string]semver.Version, error) {
	cfg := solverConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

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
				cause: RootCause{},
			},
		},
	}

	for pkg, constraint := range cfg.environmentPackages {
		s.addIncompatibility(&Incompatibility{
			terms: map[string]Term{
				pkg: {
					pkg:               pkg,
					versionConstraint: constraint,
					positive:          false, // Incompatibility will be satisfied if the package is installed with the given constraint
				},
			},
			cause: EnvironmentPackageCause{
				Pkg:        pkg,
				Constraint: constraint,
			},
		})
	}

	next := s.rootPkg

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
			return nil, fmt.Errorf("failed to make decision: %w", err)
		}
		if done {
			break
		}
	}

	result := s.partialSolution.decisionsMap()
	delete(result, s.rootPkg)
	for pkg := range cfg.environmentPackages {
		delete(result, pkg)
	}
	return result, nil
}

func (s *solver) unitPropagation(inPkg string) error {
	changed := []string{inPkg}
	for len(changed) > 0 {
		pkg := changed[0]
		changed = changed[1:]

		for i := len(s.incompatibilities) - 1; i >= 0; i-- {
			currentIncompatibility := s.incompatibilities[i]
			if _, hasPkg := currentIncompatibility.terms[pkg]; !hasPkg {
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
				break
			} else if rel == setRelationAlmostSatisfied {
				s.partialSolution.add(t.Negate(), currentIncompatibility)
				changed = append(changed, t.pkg)
			}
		}
	}
	return nil
}

func (s *solver) conflictResolution(fromIncompatibility *Incompatibility) (*Incompatibility, error) {
	incompatibilityChanged := false
	for {
		if s.isIncompatibilityTerminal(fromIncompatibility) {
			return nil, SolvingError{cause: fromIncompatibility, rootPkg: s.rootPkg}
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

		if len(priorCause.terms) > 1 {
			if rootTerm, ok := priorCause.terms[s.rootPkg]; ok && rootTerm.positive {
				delete(priorCause.terms, s.rootPkg)
			}
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
		if errors.Is(err, ErrPackageNotFound) {
			s.addIncompatibility(&Incompatibility{
				terms: map[string]Term{pkg: {pkg: pkg, versionConstraint: semver.AnyConstraint, positive: true}},
				cause: PackageNotFoundCause{Pkg: pkg},
			})
			return pkg, false, nil
		}
		return pkg, false, fmt.Errorf("failed to get package versions: %w", err)
	}

	if len(versions) == 0 {
		s.addIncompatibility(&Incompatibility{
			terms: map[string]Term{pkg: {pkg: pkg, versionConstraint: semver.AnyConstraint, positive: true}},
			cause: NoVersionsCause{Pkg: pkg, Constraint: semver.AnyConstraint},
		})
		return pkg, false, nil
	}

	// Sort versions in ascending order if not already sorted
	if !slices.IsSortedFunc(versions, func(a, b PackageVersion) int { return a.Version.Compare(b.Version) }) {
		versions = slices.Clone(versions)
		slices.SortFunc(versions, func(a, b PackageVersion) int {
			return a.Version.Compare(b.Version)
		})
	}

	allVersions := make([]semver.Version, 0, len(versions))
	for _, v := range versions {
		allVersions = append(allVersions, v.Version)
	}

	var forbiddenVersions []PackageVersion
	var allowedVersions []PackageVersion
	for _, v := range versions {
		if v.ForbiddenReason != "" {
			forbiddenVersions = append(forbiddenVersions, v)
		} else {
			allowedVersions = append(allowedVersions, v)
		}
	}

	versionsForbiddenBy := make(map[string][]semver.Version)
	for _, v := range forbiddenVersions {
		versionsForbiddenBy[v.ForbiddenReason] = append(versionsForbiddenBy[v.ForbiddenReason], v.Version)
	}

	versionsForbiddenByKeys := make([]string, 0, len(versionsForbiddenBy))
	for cause := range versionsForbiddenBy {
		versionsForbiddenByKeys = append(versionsForbiddenByKeys, cause)
	}
	slices.Sort(versionsForbiddenByKeys)

	for _, cause := range versionsForbiddenByKeys {
		versions := versionsForbiddenBy[cause]
		pkgRange := semver.NewConstraintFromVersionSubset(versions, allVersions)
		s.addIncompatibility(&Incompatibility{
			terms: map[string]Term{pkg: {pkg: pkg, versionConstraint: pkgRange, positive: true}},
			cause: PackageVersionForbiddenCause{Reason: cause, Pkg: pkg, PkgRange: pkgRange},
		})
	}

	availableVersions := make([]semver.Version, 0, len(allowedVersions))
	for _, v := range allowedVersions {
		availableVersions = append(availableVersions, v.Version)
	}

	var compatibleVersions []semver.Version
	for _, v := range availableVersions {
		if t.versionConstraint.Contains(v) {
			compatibleVersions = append(compatibleVersions, v)
		}
	}

	if len(compatibleVersions) == 0 {
		hasForbiddenMatching := slices.ContainsFunc(allVersions, func(v semver.Version) bool {
			return t.versionConstraint.Contains(v)
		})
		if hasForbiddenMatching {
			// We already added incompatibilities for forbidden versions above,
			// and we shouldn't consider this case as a "no versions" case,
			// so those incompatibilities are used to explain the conflict.
			return pkg, false, nil
		}
		s.addIncompatibility(&Incompatibility{
			terms: map[string]Term{pkg: *t},
			cause: NoVersionsCause{Pkg: pkg, Constraint: t.Constraint()},
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
		pkgRange := semver.NewConstraintFromVersionSubset(versionsWithThisDependency, allVersions)
		s.addIncompatibility(&Incompatibility{
			terms: map[string]Term{
				pkg: {
					pkg:               pkg,
					versionConstraint: pkgRange,
					positive:          true,
				},
				dep: {
					pkg:               dep,
					versionConstraint: constraint,
				},
			},
			cause: DependencyCause{Pkg: pkg, PkgRange: pkgRange, Target: dep, Constraint: constraint, Optional: false},
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
		pkgRange := semver.NewConstraintFromVersionSubset(versionsWithThisDependency, allVersions)
		s.addIncompatibility(&Incompatibility{
			terms: map[string]Term{
				pkg: {
					pkg:               pkg,
					versionConstraint: pkgRange,
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
			cause: DependencyCause{Pkg: pkg, PkgRange: pkgRange, Target: dep, Constraint: constraint, Optional: true},
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
