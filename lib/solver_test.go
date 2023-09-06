package lib

import (
	"github.com/mircearoata/pubgrub-go/lib/version"
	"github.com/pkg/errors"
	"maps"
	"testing"
)

type mockSource struct {
	packages map[string][]mockPackageVersion
}

type mockPackageVersion struct {
	version version.Version
	deps    map[string]version.Constraint
}

func (s mockSource) GetPackageVersions(pkg string) ([]version.Version, error) {
	if _, ok := s.packages[pkg]; !ok {
		return nil, errors.New("package not found")
	}
	var result []version.Version
	for _, v := range s.packages[pkg] {
		result = append(result, v.version)
	}
	return result, nil
}

func (s mockSource) GetPackageVersionsSatisfying(pkg string, constraint version.Constraint) ([]version.Version, error) {
	if _, ok := s.packages[pkg]; !ok {
		return nil, errors.New("package not found")
	}
	var result []version.Version
	for _, v := range s.packages[pkg] {
		if constraint.Contains(v.version) {
			result = append(result, v.version)
		}
	}
	return result, nil
}

func (s mockSource) GetPackageVersionDependencies(pkg string, version version.Version) (map[string]version.Constraint, error) {
	if _, ok := s.packages[pkg]; !ok {
		return nil, errors.New("package not found")
	}
	for _, v := range s.packages[pkg] {
		if v.version.Compare(version) == 0 {
			return v.deps, nil
		}
	}
	return nil, errors.New("version not found")
}

func newVersion(v string) version.Version {
	result, _ := version.NewVersion(v)
	return *result
}

func newConstraint(c string) version.Constraint {
	result, _ := version.NewConstraint(c)
	return result
}

func TestSolver_ConflictResolutionWithPartialSatisfier(t *testing.T) {
	source := mockSource{
		packages: map[string][]mockPackageVersion{
			"$$root$$": {
				{
					version: newVersion("1.0.0"),
					deps: map[string]version.Constraint{
						"foo":    newConstraint("^1.0.0"),
						"target": newConstraint("^2.0.0"),
					},
				},
			},
			"foo": {
				{
					version: newVersion("1.1.0"),
					deps: map[string]version.Constraint{
						"left":  newConstraint("^1.0.0"),
						"right": newConstraint("^1.0.0"),
					},
				},
				{
					version: newVersion("1.0.0"),
				},
			},
			"left": {
				{
					version: newVersion("1.0.0"),
					deps: map[string]version.Constraint{
						"shared": newConstraint(">=1.0.0"),
					},
				},
			},
			"right": {
				{
					version: newVersion("1.0.0"),
					deps: map[string]version.Constraint{
						"shared": newConstraint("<2.0.0"),
					},
				},
			},
			"shared": {
				{
					version: newVersion("2.0.0"),
				},
				{
					version: newVersion("1.0.0"),
					deps: map[string]version.Constraint{
						"target": newConstraint("^1.0.0"),
					},
				},
			},
			"target": {
				{
					version: newVersion("2.0.0"),
				},
				{
					version: newVersion("1.0.0"),
				},
			},
		},
	}

	result, err := Solve(source, "$$root$$")
	if err != nil {
		t.Fatal(err)
	}
	delete(result, "$$root$$")
	expected := map[string]version.Version{
		"foo":    newVersion("1.0.0"),
		"target": newVersion("2.0.0"),
	}
	if !maps.EqualFunc(result, expected, func(v version.Version, v2 version.Version) bool {
		return v.Compare(v2) == 0
	}) {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}

func TestSolver_LinearErrorReporting(t *testing.T) {
	source := mockSource{
		packages: map[string][]mockPackageVersion{
			"$$root$$": {
				{
					version: newVersion("1.0.0"),
					deps: map[string]version.Constraint{
						"foo": newConstraint("^1.0.0"),
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					version: newVersion("1.0.0"),
					deps: map[string]version.Constraint{
						"bar": newConstraint("^2.0.0"),
					},
				},
			},
			"bar": {
				{
					version: newVersion("2.0.0"),
					deps: map[string]version.Constraint{
						"baz": newConstraint("^3.0.0"),
					},
				},
			},
			"baz": {
				{
					version: newVersion("1.0.0"),
				},
				{
					version: newVersion("3.0.0"),
				},
			},
		},
	}

	result, err := Solve(source, "$$root$$")
	if err == nil {
		delete(result, "$$root$$")
		t.Fatalf("expected error, but resolved successfully: %s", result)
	}
	expected := "Because every version of foo depends on bar \"^2.0.0\" and every version of bar depends on baz \"^3.0.0\", every version of foo depends on baz \"^3.0.0\".\nAnd because installing baz \"^1.0.0\", resolving failed."
	if err.Error() != expected {
		t.Fatalf("expected error\n%s\n\ngot\n%s", expected, err.Error())
	}
}

func TestSolver_BranchingErrorReporting(t *testing.T) {
	source := mockSource{
		packages: map[string][]mockPackageVersion{
			"$$root$$": {
				{
					version: newVersion("1.0.0"),
					deps: map[string]version.Constraint{
						"foo": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					version: newVersion("1.0.0"),
					deps: map[string]version.Constraint{
						"a": newConstraint("^1.0.0"),
						"b": newConstraint("^1.0.0"),
					},
				},
				{
					version: newVersion("1.1.0"),
					deps: map[string]version.Constraint{
						"x": newConstraint("^1.0.0"),
						"y": newConstraint("^1.0.0"),
					},
				},
			},
			"a": {
				{
					version: newVersion("1.0.0"),
					deps: map[string]version.Constraint{
						"b": newConstraint("^2.0.0"),
					},
				},
			},
			"b": {
				{
					version: newVersion("1.0.0"),
				},
				{
					version: newVersion("2.0.0"),
				},
			},
			"x": {
				{
					version: newVersion("1.0.0"),
					deps: map[string]version.Constraint{
						"y": newConstraint("^2.0.0"),
					},
				},
			},
			"y": {
				{
					version: newVersion("1.0.0"),
				},
				{
					version: newVersion("2.0.0"),
				},
			},
		},
	}

	result, err := Solve(source, "$$root$$")
	if err == nil {
		delete(result, "$$root$$")
		t.Fatalf("expected error, but resolved successfully: %s", result)
	}
	expected := "Because every version of a depends on b \"^2.0.0\" and foo \"<1.1.0\" depends on a \"^1.0.0\", foo \"<1.1.0\" depends on b \"^2.0.0\".\n1. And because foo \"<1.1.0\" depends on b \"^1.0.0\", foo \"<1.1.0\" is forbidden.\n\nBecause every version of x depends on y \"^2.0.0\" and foo \">=1.1.0\" depends on x \"^1.0.0\", foo \">=1.1.0\" depends on y \"^2.0.0\".\n2. And because foo \">=1.1.0\" depends on y \"^1.0.0\", foo \">=1.1.0\" is forbidden.\nAnd because foo \"<1.1.0\" is forbidden (1), foo is forbidden.\nAnd because installing foo \"^1.0.0\", resolving failed."
	if err.Error() != expected {
		t.Fatalf("expected error\n%s\n\ngot\n%s", expected, err.Error())
	}
}
