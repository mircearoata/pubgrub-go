package pubgrub

import (
	"errors"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/mircearoata/pubgrub-go/pubgrub/semver"
)

type mockSource struct {
	packages map[string][]PackageVersion
}

func (s mockSource) GetPackageVersions(pkg string) ([]PackageVersion, error) {
	if v, ok := s.packages[pkg]; ok {
		return v, nil
	}
	return nil, ErrPackageNotFound
}

func (s mockSource) PickVersion(_ string, versions []semver.Version) semver.Version {
	return versions[len(versions)-1]
}

func newVersion(v string) semver.Version {
	result, _ := semver.NewVersion(v)
	return result
}

func newConstraint(c string) semver.Constraint {
	result, _ := semver.NewConstraint(c)
	return result
}

func TestSolver_ConflictResolutionWithPartialSatisfier(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"foo":    newConstraint("^1.0.0"),
						"target": newConstraint("^2.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.1.0"),
					Dependencies: map[string]semver.Constraint{
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
					Dependencies: map[string]semver.Constraint{
						"shared": newConstraint(">=1.0.0"),
					},
				},
			},
			"right": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
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
					Dependencies: map[string]semver.Constraint{
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
	testza.AssertNoError(t, err)

	expected := map[string]semver.Version{
		"foo":    newVersion("1.0.0"),
		"target": newVersion("2.0.0"),
	}
	testza.AssertEqual(t, expected, result)
}

func TestSolver_LinearErrorReporting(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"foo": newConstraint("^1.0.0"),
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"bar": newConstraint("^2.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("2.0.0"),
					Dependencies: map[string]semver.Constraint{
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
	testza.AssertNil(t, result)
	expected := "Because every version of foo depends on bar \"^2.0.0\" which depends on baz \"^3.0.0\", every version of foo requires baz \"^3.0.0\".\nSo, because installing both baz \"^1.0.0\" and foo \"^1.0.0\", version solving failed."
	testza.AssertEqual(t, expected, err.Error())
}

func TestSolver_BranchingErrorReporting(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"foo": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"a": newConstraint("^1.0.0"),
						"b": newConstraint("^1.0.0"),
					},
				},
				{
					Version: newVersion("1.1.0"),
					Dependencies: map[string]semver.Constraint{
						"x": newConstraint("^1.0.0"),
						"y": newConstraint("^1.0.0"),
					},
				},
			},
			"a": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
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
					Dependencies: map[string]semver.Constraint{
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
	testza.AssertNil(t, result)
	expected := "   Because foo \"<1.1.0\" depends on a \"^1.0.0\" which depends on b \"^2.0.0\", foo \"<1.1.0\" requires b \"^2.0.0\".\n1. So, because foo \"<1.1.0\" depends on b \"^1.0.0\", foo \"<1.1.0\" is forbidden.\n\n   Because foo \">=1.1.0\" depends on x \"^1.0.0\" which depends on y \"^2.0.0\", foo \">=1.1.0\" requires y \"^2.0.0\".\n   And because foo \">=1.1.0\" depends on y \"^1.0.0\", foo \">=1.1.0\" is forbidden.\n   And because foo \"<1.1.0\" is forbidden (1), every version of foo is forbidden.\n   So, because installing foo \"^1.0.0\", version solving failed."
	testza.AssertEqual(t, expected, err.Error())
}

func TestSolver_OptionalDependencies_NoOptional(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"foo": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					OptionalDependencies: map[string]semver.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
				{
					Version: newVersion("1.0.1"),
					Dependencies: map[string]semver.Constraint{
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
	testza.AssertNoError(t, err)

	expected := map[string]semver.Version{
		"foo": newVersion("1.0.0"),
	}
	testza.AssertEqual(t, expected, result)
}

func TestSolver_OptionalDependencies_CompatibleVersion(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"foo": newConstraint("^1.0.0"),
						"bar": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					OptionalDependencies: map[string]semver.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
				{
					Version: newVersion("1.0.1"),
					Dependencies: map[string]semver.Constraint{
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
	testza.AssertNoError(t, err)

	expected := map[string]semver.Version{
		"foo": newVersion("1.0.0"),
		"bar": newVersion("1.0.0"),
		"baz": newVersion("1.0.0"),
	}
	testza.AssertEqual(t, expected, result)
}

func TestSolver_OptionalDependencies_Error(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"foo": newConstraint("^1.0.0"),
						"bar": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					OptionalDependencies: map[string]semver.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
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
	testza.AssertNil(t, result)
	expected := "Because every version of bar depends on baz \"^2.0.0\" and every version of foo optionally depends on baz \"^1.0.0\", every version of bar is incompatible with foo.\nSo, because installing both bar \"^1.0.0\" and foo \"^1.0.0\", version solving failed."
	testza.AssertEqual(t, expected, err.Error())
}

func TestSolver_DependsOnBoth(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"parent": newConstraint("^1.0.0"),
					},
				},
			},
			"parent": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"my_lib": newConstraint("^1.0.0"),
					},
				},
			},
			"my_lib": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"bar": newConstraint("^1.0.0"),
						"foo": newConstraint("^1.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"shared": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"shared": newConstraint("^2.0.0"),
					},
				},
			},
			"shared": {
				{Version: newVersion("1.0.0")},
				{Version: newVersion("2.0.0")},
			},
		},
	}

	result, err := Solve(source, "$$root$$")
	testza.AssertNil(t, result)
	expected := "Because every version of foo depends on shared \"^2.0.0\" and every version of bar depends on shared \"^1.0.0\", every version of bar is incompatible with foo.\nAnd because every version of my_lib depends on both bar \"^1.0.0\" and foo \"^1.0.0\", every version of my_lib is forbidden.\nSo, because installing parent \"^1.0.0\" which depends on my_lib \"^1.0.0\", version solving failed."
	testza.AssertEqual(t, expected, err.Error())
}

func TestSolver_WhichDependsOn(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"foo": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"bar": newConstraint("^1.0.0"),
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"baz": newConstraint("^2.0.0"),
					},
				},
			},
			"baz": {
				{Version: newVersion("1.0.0")},
				{Version: newVersion("2.0.0")},
			},
		},
	}

	result, err := Solve(source, "$$root$$")
	testza.AssertNil(t, result)
	expected := "Because every version of foo depends on bar \"^1.0.0\" which depends on baz \"^2.0.0\", every version of foo requires baz \"^2.0.0\".\nSo, because installing foo \"^1.0.0\" which depends on baz \"^1.0.0\", version solving failed."
	testza.AssertEqual(t, expected, err.Error())
}

func TestSolver_WhichIsForbidden(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"foo": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"bar": newConstraint("^1.0.0"),
					},
				},
			},
			"bar": {},
		},
	}

	result, err := Solve(source, "$$root$$")
	testza.AssertNil(t, result)
	expected := "Because every version of foo depends on bar \"^1.0.0\" which matches no versions, every version of foo is forbidden.\nSo, because installing foo \"^1.0.0\", version solving failed."
	testza.AssertEqual(t, expected, err.Error())
}

type customPkgFormatter struct{}

func (f customPkgFormatter) FormatPackage(pkg string) string {
	if pkg == "foo" {
		return "foo2"
	}
	return pkg
}

type customConstraintFormatter struct{}

func (f customConstraintFormatter) FormatConstraint(_ string, c semver.Constraint) string {
	return "v" + c.String()
}

func TestSolver_Formatters(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"foo": newConstraint("^1.0.0"),
						"bar": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {
				{
					Version: newVersion("1.0.0"),
					OptionalDependencies: map[string]semver.Constraint{
						"baz": newConstraint("^1.0.0"),
					},
				},
			},
			"bar": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"baz": newConstraint("^2.0.0"),
					},
				},
			},
			"baz": {
				{Version: newVersion("1.0.0")},
				{Version: newVersion("2.0.0")},
			},
		},
	}

	result, err := Solve(source, "$$root$$")
	testza.AssertNil(t, result)
	var solverErr SolvingError
	testza.AssertTrue(t, errors.As(err, &solverErr))

	textReporter := NewStandardTextReporter().
		WithTermStringer(
			NewStandardTermStringer().
				WithPackageFormatter(customPkgFormatter{}).
				WithConstraintFormatter(customConstraintFormatter{}),
		)
	rendered := textReporter.Render(solverErr.Report())
	expected := "Because every version of bar depends on baz \"v^2.0.0\" and every version of foo2 optionally depends on baz \"v^1.0.0\", every version of bar is incompatible with foo2.\nSo, because installing both bar \"v^1.0.0\" and foo2 \"v^1.0.0\", version solving failed."
	testza.AssertEqual(t, expected, rendered)
}

func TestSolver_CustomStrings(t *testing.T) {
	t.Parallel()

	source := mockSource{
		packages: map[string][]PackageVersion{
			"$$root$$": {
				{
					Version: newVersion("1.0.0"),
					Dependencies: map[string]semver.Constraint{
						"foo": newConstraint("^1.0.0"),
					},
				},
			},
			"foo": {},
		},
	}

	result, err := Solve(source, "$$root$$")
	testza.AssertNil(t, result)
	var solverErr SolvingError
	testza.AssertTrue(t, errors.As(err, &solverErr))

	customCauses := DefaultCauseStrings
	customCauses.Because = "(Because) %s, %s."

	customIncompats := DefaultIncompatibilityStrings
	customIncompats.Installing = "(install) %s"
	customIncompats.ResolvingFailed = "(failed)"

	reporter := NewStandardTextReporter().
		WithStrings(customCauses).
		WithIncompatibilityStringer(
			NewStandardIncompatibilityStringer().
				WithStrings(customIncompats),
		)

	rendered := reporter.Render(solverErr.Report())
	expected := "(Because) (install) foo \"^1.0.0\" which matches no versions, (failed)."
	testza.AssertEqual(t, expected, rendered)
}

func TestSolver_PackageNotFound(t *testing.T) {
	t.Parallel()

	t.Run("direct", func(t *testing.T) {
		t.Parallel()

		source := mockSource{
			packages: map[string][]PackageVersion{
				"$$root$$": {
					{
						Version: newVersion("1.0.0"),
						Dependencies: map[string]semver.Constraint{
							"foo": newConstraint("^1.0.0"),
						},
					},
				},
			},
		}

		result, err := Solve(source, "$$root$$")
		testza.AssertNil(t, result)
		expected := "Because installing foo \"^1.0.0\" which could not be found, version solving failed."
		testza.AssertEqual(t, expected, err.Error())
	})

	t.Run("transitive", func(t *testing.T) {
		t.Parallel()

		source := mockSource{
			packages: map[string][]PackageVersion{
				"$$root$$": {
					{
						Version: newVersion("1.0.0"),
						Dependencies: map[string]semver.Constraint{
							"foo": newConstraint("^1.0.0"),
						},
					},
				},
				"foo": {
					{
						Version: newVersion("1.0.0"),
						Dependencies: map[string]semver.Constraint{
							"bar": newConstraint("^1.0.0"),
						},
					},
				},
			},
		}

		result, err := Solve(source, "$$root$$")
		testza.AssertNil(t, result)
		expected := "Because every version of foo depends on bar \"^1.0.0\" which could not be found, every version of foo is forbidden.\nSo, because installing foo \"^1.0.0\", version solving failed."
		testza.AssertEqual(t, expected, err.Error())
	})
}

func TestSolver_ForbiddenVersion(t *testing.T) {
	t.Parallel()

	t.Run("none allowed", func(t *testing.T) {
		t.Parallel()

		t.Run("direct", func(t *testing.T) {
			t.Parallel()

			source := mockSource{
				packages: map[string][]PackageVersion{
					"$$root$$": {
						{
							Version: newVersion("1.0.0"),
							Dependencies: map[string]semver.Constraint{
								"foo": newConstraint("^1.0.0"),
							},
						},
					},
					"foo": {
						{
							Version:         newVersion("1.0.0"),
							ForbiddenReason: "is broken",
						},
					},
				},
			}

			result, err := Solve(source, "$$root$$")
			testza.AssertNil(t, result)
			expected := "Because installing foo \"^1.0.0\" which is broken, version solving failed."
			testza.AssertEqual(t, expected, err.Error())
		})

		t.Run("transitive", func(t *testing.T) {
			t.Parallel()

			source := mockSource{
				packages: map[string][]PackageVersion{
					"$$root$$": {
						{
							Version: newVersion("1.0.0"),
							Dependencies: map[string]semver.Constraint{
								"foo": newConstraint("^1.0.0"),
							},
						},
					},
					"foo": {
						{
							Version: newVersion("1.0.0"),
							Dependencies: map[string]semver.Constraint{
								"bar": newConstraint("^1.0.0"),
							},
						},
					},
					"bar": {
						{
							Version:         newVersion("1.0.0"),
							ForbiddenReason: "is broken",
						},
					},
				},
			}

			result, err := Solve(source, "$$root$$")
			testza.AssertNil(t, result)
			expected := "Because every version of foo depends on bar \"^1.0.0\" which is broken, every version of foo is forbidden.\nSo, because installing foo \"^1.0.0\", version solving failed."
			testza.AssertEqual(t, expected, err.Error())
		})

		t.Run("requested non existent", func(t *testing.T) {
			t.Parallel()

			source := mockSource{
				packages: map[string][]PackageVersion{
					"$$root$$": {
						{
							Version: newVersion("1.0.0"),
							Dependencies: map[string]semver.Constraint{
								"foo": newConstraint("2.0.0"),
							},
						},
					},
					"foo": {
						{
							Version:         newVersion("1.0.0"),
							ForbiddenReason: "is broken",
						},
					},
				},
			}

			result, err := Solve(source, "$$root$$")
			testza.AssertNil(t, result)
			expected := "Because installing foo \"2.0.0\" which matches no versions, version solving failed."
			testza.AssertEqual(t, expected, err.Error())
		})
	})

	t.Run("some allowed", func(t *testing.T) {
		t.Parallel()

		t.Run("requested forbidden", func(t *testing.T) {
			t.Parallel()

			source := mockSource{
				packages: map[string][]PackageVersion{
					"$$root$$": {
						{
							Version: newVersion("1.0.0"),
							Dependencies: map[string]semver.Constraint{
								"foo": newConstraint(">=2.0.0"),
							},
						},
					},
					"foo": {
						{
							Version: newVersion("1.0.0"),
						},
						{
							Version:         newVersion("2.0.0"),
							ForbiddenReason: "is deprecated",
						},
					},
				},
			}

			result, err := Solve(source, "$$root$$")
			testza.AssertNil(t, result)
			expected := "Because installing foo \">=2.0.0\" which is deprecated, version solving failed."
			testza.AssertEqual(t, expected, err.Error())
		})

		t.Run("requested allowed", func(t *testing.T) {
			t.Parallel()

			source := mockSource{
				packages: map[string][]PackageVersion{
					"$$root$$": {
						{
							Version: newVersion("1.0.0"),
							Dependencies: map[string]semver.Constraint{
								"foo": newConstraint(">=1.0.0"),
							},
						},
					},
					"foo": {
						{
							Version:         newVersion("1.0.0"),
							ForbiddenReason: "is deprecated",
						},
						{
							Version: newVersion("2.0.0"),
						},
					},
				},
			}

			result, err := Solve(source, "$$root$$")
			testza.AssertNoError(t, err)
			testza.AssertEqual(t, map[string]semver.Version{"foo": newVersion("2.0.0")}, result)
		})

		t.Run("requested non existent", func(t *testing.T) {
			t.Parallel()

			source := mockSource{
				packages: map[string][]PackageVersion{
					"$$root$$": {
						{
							Version: newVersion("1.0.0"),
							Dependencies: map[string]semver.Constraint{
								"foo": newConstraint("3.0.0"),
							},
						},
					},
					"foo": {
						{
							Version:         newVersion("1.0.0"),
							ForbiddenReason: "is deprecated",
						},
						{
							Version: newVersion("2.0.0"),
						},
					},
				},
			}

			result, err := Solve(source, "$$root$$")
			testza.AssertNil(t, result)
			expected := "Because installing foo \"3.0.0\" which matches no versions, version solving failed."
			testza.AssertEqual(t, expected, err.Error())
		})
	})

	t.Run("multiple reasons", func(t *testing.T) {
		t.Parallel()

		source := mockSource{
			packages: map[string][]PackageVersion{
				"$$root$$": {
					{
						Version: newVersion("1.0.0"),
						Dependencies: map[string]semver.Constraint{
							"foo": newConstraint(">=1.0.0"),
						},
					},
				},
				"foo": {
					{
						Version:         newVersion("1.0.0"),
						ForbiddenReason: "is broken",
					},
					{
						Version:         newVersion("1.1.0"),
						ForbiddenReason: "is broken",
					},
					{
						Version:         newVersion("2.0.0"),
						ForbiddenReason: "is deprecated",
					},
					{
						Version:         newVersion("2.1.0"),
						ForbiddenReason: "is deprecated",
					},
					{
						Version:         newVersion("3.0.0"),
						ForbiddenReason: "is broken",
					},
				},
			},
		}

		result, err := Solve(source, "$$root$$")
		testza.AssertNil(t, result)
		expected := "Because foo \"<2.0.0 || >=3.0.0\" is broken and foo \"^2.0.0\" is deprecated, every version of foo is forbidden.\nSo, because installing foo \">=1.0.0\", version solving failed."
		testza.AssertEqual(t, expected, err.Error())
	})
}

func TestSolver_EnvironmentConstraints(t *testing.T) {
	t.Parallel()

	t.Run("unsatisfied", func(t *testing.T) {
		t.Parallel()

		source := mockSource{
			packages: map[string][]PackageVersion{
				"$$root$$": {
					{
						Version: newVersion("1.0.0"),
						Dependencies: map[string]semver.Constraint{
							"foo": newConstraint("^1.0.0"),
						},
					},
				},
				"foo": {
					{
						Version: newVersion("1.0.0"),
						Dependencies: map[string]semver.Constraint{
							"go": newConstraint(">=2.0.0"),
						},
					},
				},
				"go": {
					{
						Version: newVersion("2.0.0"),
					},
				},
			},
		}

		result, err := Solve(
			source,
			"$$root$$",
			WithEnvironmentPackages(map[string]semver.Constraint{
				"go": newConstraint("1.0.0"),
			}),
		)
		testza.AssertNil(t, result)
		expected := "Because every version of foo depends on go \">=2.0.0\" and go \"1.0.0\" is installed, every version of foo is forbidden.\nSo, because installing foo \"^1.0.0\", version solving failed."
		testza.AssertEqual(t, expected, err.Error())
	})

	t.Run("satisfied", func(t *testing.T) {
		t.Parallel()

		source := mockSource{
			packages: map[string][]PackageVersion{
				"$$root$$": {
					{
						Version: newVersion("1.0.0"),
						Dependencies: map[string]semver.Constraint{
							"foo": newConstraint("^1.0.0"),
						},
					},
				},
				"foo": {
					{
						Version: newVersion("1.0.0"),
						Dependencies: map[string]semver.Constraint{
							"go": newConstraint("1.0.0"),
						},
					},
				},
				"go": {
					{
						Version: newVersion("1.0.0"),
					},
				},
			},
		}

		result, err := Solve(
			source,
			"$$root$$",
			WithEnvironmentPackages(map[string]semver.Constraint{
				"go": newConstraint("1.0.0"),
			}),
		)
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, map[string]semver.Version{
			"foo": newVersion("1.0.0"),
		}, result)
	})
}
