package semver

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestNewConstraint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		constraint string
		expected   Constraint
	}{
		// already canonical
		// doesn't need too much testing on basic parsing, since that's tested in version_range_test.go
		{"1.2.3", Constraint{[]versionRange{{&Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, &Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, true, true, "1.2.3"}}, "1.2.3"}},
		{"1.2.3 || 2.3.4", Constraint{[]versionRange{{&Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, &Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, true, true, "1.2.3"}, {&Version{major: 2, minor: 3, patch: 4, raw: "2.3.4"}, &Version{major: 2, minor: 3, patch: 4, raw: "2.3.4"}, true, true, "2.3.4"}}, "1.2.3 || 2.3.4"}},
		// canonicalization
		{"1.2.3 || 1.2.3", Constraint{[]versionRange{{&Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, &Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, true, true, "1.2.3"}}, "1.2.3 || 1.2.3"}},
		{"<1.2.3 || <=1.2.3", Constraint{[]versionRange{{nil, &Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, false, true, "<=1.2.3"}}, "<1.2.3 || <=1.2.3"}},
		{"<=1.2.3 || <1.2.3", Constraint{[]versionRange{{nil, &Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, false, true, "<=1.2.3"}}, "<=1.2.3 || <1.2.3"}},
		{">1.2.3 || >=1.2.3", Constraint{[]versionRange{{&Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, nil, true, false, ">=1.2.3"}}, ">1.2.3 || >=1.2.3"}},
		{">=1.2.3 || >1.2.3", Constraint{[]versionRange{{&Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, nil, true, false, ">=1.2.3"}}, ">=1.2.3 || >1.2.3"}},
		{">=1.2.3 || >=1.2.4", Constraint{[]versionRange{{&Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, nil, true, false, ">=1.2.3"}}, ">=1.2.3 || >=1.2.4"}},
		{"<1.0.0 || >1.0.0", Constraint{[]versionRange{{nil, &Version{major: 1, minor: 0, patch: 0, raw: "1.0.0"}, false, false, "<1.0.0"}, {&Version{major: 1, minor: 0, patch: 0, raw: "1.0.0"}, nil, false, false, ">1.0.0"}}, "<1.0.0 || >1.0.0"}},
		{"<1.0.0 || >=1.0.0", Constraint{[]versionRange{rangeAny}, "<1.0.0 || >=1.0.0"}}, // this can be canonicalized to "any" because pre-releases are not allowed in those
		{"<1.0.0 || >=1.0.0-alpha", Constraint{[]versionRange{rangeAny}, "<1.0.0 || >=1.0.0-alpha"}},
		{"<1.0.0-0 || >=1.0.0", Constraint{[]versionRange{{nil, &Version{1, 0, 0, []string{"0"}, nil, "1.0.0-0"}, false, false, "<1.0.0-0"}, {&Version{1, 0, 0, nil, nil, "1.0.0"}, nil, true, false, ">=1.0.0"}}, "<1.0.0-0 || >=1.0.0"}},
		{">=1.2.3-alpha <=1.5.0 || >=1.4.0 <2.0.0", Constraint{[]versionRange{{&Version{1, 2, 3, []string{"alpha"}, nil, "1.2.3-alpha"}, &Version{2, 0, 0, nil, nil, "2.0.0"}, true, false, "^1.2.3-alpha"}}, ">=1.2.3-alpha <=1.5.0 || >=1.4.0 <2.0.0"}},
		{">=1.0.0 <=1.2.3-alpha || >=0.5.0", Constraint{[]versionRange{{&Version{0, 5, 0, nil, nil, "0.5.0"}, nil, true, false, ">=0.5.0"}}, ">=1.0.0 <=1.2.3-alpha || >=0.5.0"}},
		{">=1.0.0 <=1.2.3-alpha || >=1.1.0", Constraint{[]versionRange{{&Version{1, 0, 0, nil, nil, "1.0.0"}, nil, true, false, ">=1.0.0"}}, ">=1.0.0 <=1.2.3-alpha || >=1.1.0"}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.constraint, func(t *testing.T) {
			t.Parallel()
			actual, err := NewConstraint(test.constraint)
			testza.AssertNoError(t, err, "NewConstraint(%s)", test.constraint)
			testza.AssertEqual(t, test.expected, actual, "NewConstraint(%s)", test.constraint)
			testza.AssertEqual(t, test.constraint, actual.RawString(), "NewConstraint(%s).RawString()", test.constraint)
		})
	}
}

func TestConstraint_Intersect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		c1       Constraint
		c2       Constraint
		expected Constraint
	}{
		// equal
		{Constraint{[]versionRange{{&Version{1, 0, 0, nil, nil, "1.0.0"}, &Version{1, 0, 0, nil, nil, "1.0.0"}, true, true, "1.0.0"}}, "1.0.0"}, Constraint{[]versionRange{{&Version{1, 0, 0, nil, nil, "1.0.0"}, &Version{1, 0, 0, nil, nil, "1.0.0"}, true, true, "1.0.0"}}, "1.0.0"}, Constraint{[]versionRange{{&Version{1, 0, 0, nil, nil, "1.0.0"}, &Version{1, 0, 0, nil, nil, "1.0.0"}, true, true, "1.0.0"}}, "1.0.0"}},
		// intersecting
		{Constraint{[]versionRange{{&Version{1, 0, 0, nil, nil, "1.0.0"}, nil, true, false, ">=1.0.0"}}, ">=1.0.0"}, Constraint{[]versionRange{{nil, &Version{1, 0, 0, nil, nil, "1.0.0"}, false, true, "<=1.0.0"}}, "<=1.0.0"}, Constraint{[]versionRange{{&Version{1, 0, 0, nil, nil, "1.0.0"}, &Version{1, 0, 0, nil, nil, "1.0.0"}, true, true, "1.0.0"}}, "1.0.0"}},
		// intersecting with pre-releases
		{Constraint{[]versionRange{{&Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, nil, true, false, ">=1.0.0-alpha"}}, ">=1.0.0-alpha"}, Constraint{[]versionRange{{nil, &Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, false, true, "<=1.0.0-alpha"}}, "<=1.0.0-alpha"}, Constraint{[]versionRange{{&Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, &Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, true, true, "1.0.0-alpha"}}, "1.0.0-alpha"}},
		// non-intersecting
		{Constraint{[]versionRange{{&Version{1, 0, 0, nil, nil, "1.0.0"}, nil, true, false, ">=1.0.0"}}, ">=1.0.0"}, Constraint{[]versionRange{{nil, &Version{1, 0, 0, nil, nil, "1.0.0"}, false, false, "<1.0.0"}}, "<1.0.0"}, Constraint{nil, ""}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.c1.String()+" intersect "+test.c2.String(), func(t *testing.T) {
			t.Parallel()
			actual := test.c1.canonical().Intersect(test.c2.canonical())
			testza.AssertEqual(t, test.expected, actual, "Intersect(%s, %s)", test.c1, test.c2)
		})
	}
}
