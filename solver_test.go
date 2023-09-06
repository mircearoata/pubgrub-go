package pubgrub

import (
	"github.com/mircearoata/pubgrub-go/version"
	"github.com/pkg/errors"
	"maps"
	"testing"
)

type mockSource struct {
	packages map[string][]PackageVersion
}

func (s mockSource) GetPackageVersions(pkg string) ([]PackageVersion, error) {
	if v, ok := s.packages[pkg]; ok {
		return v, nil
	}
	return nil, errors.New("package not found")
}

func (s mockSource) PickVersion(_ string, versions []version.Version) version.Version {
	return versions[len(versions)-1]
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
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"foo":    newConstraint("^1.0.0"),
						"target": newConstraint("^2.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.1.0"),
					Dependencies: map[string]version.Constraint{
						"left":  newConstraint("^1.0.0"),
						"right": newConstraint("^1.0.0"),
					},
				},
				{
					Version: newVersion("1.0.0"),
				},
			},
			"left": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"shared": newConstraint(">=1.0.0"),
					},
				},
			},
			"right": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"shared": newConstraint("<2.0.0"),
					},
				},
			},
			"shared": {
				{
					Version: newVersion("2.0.0"),
				},
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"target": newConstraint("^1.0.0"),
					},
				},
			},
			"target": {
				{
					Version: newVersion("2.0.0"),
				},
				{
					Version: newVersion("1.0.0"),
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
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"foo": newConstraint("^1.0.0"),
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"bar": newConstraint("^2.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("2.0.0"),
					Dependencies: map[string]version.Constraint{
						"baz": newConstraint("^3.0.0"),
					},
				},
			},
			"baz": {
				{
					Version: newVersion("1.0.0"),
				},
				{
					Version: newVersion("3.0.0"),
				},
			},
		},
	}

	result, err := Solve(source, "$$root$$")
	if err == nil {
		delete(result, "$$root$$")
		t.Fatalf("expected error, but resolved successfully: %s", result)
	}
	expected := "Because every version of foo depends on bar \"^2.0.0\" and every version of bar depends on baz \"^3.0.0\", every version of foo depends on baz \"^3.0.0\".\nSo, because installing baz \"^1.0.0\", version solving failed."
	if err.Error() != expected {
		t.Fatalf("expected error\n%s\n\ngot\n%s", expected, err.Error())
	}
}

func TestSolver_BranchingErrorReporting(t *testing.T) {
	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"foo": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"a": newConstraint("^1.0.0"),
						"b": newConstraint("^1.0.0"),
					},
				},
				{
					Version: newVersion("1.1.0"),
					Dependencies: map[string]version.Constraint{
						"x": newConstraint("^1.0.0"),
						"y": newConstraint("^1.0.0"),
					},
				},
			},
			"a": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"b": newConstraint("^2.0.0"),
					},
				},
			},
			"b": {
				{
					Version: newVersion("1.0.0"),
				},
				{
					Version: newVersion("2.0.0"),
				},
			},
			"x": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"y": newConstraint("^2.0.0"),
					},
				},
			},
			"y": {
				{
					Version: newVersion("1.0.0"),
				},
				{
					Version: newVersion("2.0.0"),
				},
			},
		},
	}

	result, err := Solve(source, "$$root$$")
	if err == nil {
		delete(result, "$$root$$")
		t.Fatalf("expected error, but resolved successfully: %s", result)
	}
	expected := "Because foo \"<1.1.0\" depends on a \"^1.0.0\" and every version of a depends on b \"^2.0.0\", foo \"<1.1.0\" depends on b \"^2.0.0\".\n1. And because foo \"<1.1.0\" depends on b \"^1.0.0\", foo \"<1.1.0\" is forbidden.\n\nBecause foo \">=1.1.0\" depends on x \"^1.0.0\" and every version of x depends on y \"^2.0.0\", foo \">=1.1.0\" depends on y \"^2.0.0\".\n2. And because foo \">=1.1.0\" depends on y \"^1.0.0\", foo \">=1.1.0\" is forbidden.\nAnd because foo \"<1.1.0\" is forbidden (1), foo is forbidden.\nAnd because installing foo \"^1.0.0\", version solving failed."
	if err.Error() != expected {
		t.Fatalf("expected error\n%s\n\ngot\n%s", expected, err.Error())
	}
}

func TestSolver_OptionalDependencies_NoOptional(t *testing.T) {
	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"foo": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					OptionalDependencies: map[string]version.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
				{
					Version: newVersion("1.0.1"),
					Dependencies: map[string]version.Constraint{
						"baz": newConstraint("^2.0.0"),
					},
				},
			},
			"baz": {
				{
					Version: newVersion("1.0.0"),
				},
				{
					Version: newVersion("2.0.0"),
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
		"foo": newVersion("1.0.0"),
	}
	if !maps.EqualFunc(result, expected, func(v version.Version, v2 version.Version) bool {
		return v.Compare(v2) == 0
	}) {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}

func TestSolver_OptionalDependencies_CompatibleVersion(t *testing.T) {
	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"foo": newConstraint("^1.0.0"),
						"bar": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					OptionalDependencies: map[string]version.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
				{
					Version: newVersion("1.0.1"),
					Dependencies: map[string]version.Constraint{
						"baz": newConstraint("^2.0.0"),
					},
				},
			},
			"baz": {
				{
					Version: newVersion("1.0.0"),
				},
				{
					Version: newVersion("2.0.0"),
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
		"foo": newVersion("1.0.0"),
		"bar": newVersion("1.0.0"),
		"baz": newVersion("1.0.0"),
	}
	if !maps.EqualFunc(result, expected, func(v version.Version, v2 version.Version) bool {
		return v.Compare(v2) == 0
	}) {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}

func TestSolver_OptionalDependencies_Error(t *testing.T) {
	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"foo": newConstraint("^1.0.0"),
						"bar": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					OptionalDependencies: map[string]version.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]version.Constraint{
						"baz": newConstraint("^2.0.0"),
					},
				},
			},
			"baz": {
				{
					Version: newVersion("1.0.0"),
				},
				{
					Version: newVersion("2.0.0"),
				},
			},
		},
	}

	result, err := Solve(source, "$$root$$")
	if err == nil {
		delete(result, "$$root$$")
		t.Fatalf("expected error, but resolved successfully: %s", result)
	}
	expected := "Because every version of bar depends on baz \"^2.0.0\" and every version of foo depends on baz \"^1.0.0\", every version of foo forbids bar.\nSo, because installing bar \"^1.0.0\", version solving failed."
	if err.Error() != expected {
		t.Fatalf("expected error\n%s\n\ngot\n%s", expected, err.Error())
	}
}
