package version

import (
	"reflect"
	"testing"
)

func TestNewConstraint(t *testing.T) {
	type test struct {
		constraint string
		expected   Constraint
	}

	var tests = []test{
		// already canonical
		// doesn't need too much testing on basic parsing, since that's tested in version_range_test.go
		{"1.2.3", Constraint{[]versionRange{{&Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, &Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, true, true, "1.2.3"}}, "1.2.3"}},
		{"1.2.3 || 2.3.4", Constraint{[]versionRange{{&Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, &Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, true, true, "1.2.3"}, {&Version{major: 2, minor: 3, patch: 4, raw: "2.3.4"}, &Version{major: 2, minor: 3, patch: 4, raw: "2.3.4"}, true, true, "2.3.4"}}, "1.2.3 || 2.3.4"}},
		// canonicalization
		{"1.2.3 || 1.2.3", Constraint{[]versionRange{{&Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, &Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, true, true, "1.2.3"}}, "1.2.3 || 1.2.3"}},
		{">=1.2.3 || >=1.2.4", Constraint{[]versionRange{{&Version{major: 1, minor: 2, patch: 3, raw: "1.2.3"}, nil, true, false, ">=1.2.3"}}, ">=1.2.3 || >=1.2.4"}},
		{"<1.0.0 || >1.0.0", Constraint{[]versionRange{{nil, &Version{major: 1, minor: 0, patch: 0, raw: "1.0.0"}, false, false, "<1.0.0"}, {&Version{major: 1, minor: 0, patch: 0, raw: "1.0.0"}, nil, false, false, ">1.0.0"}}, "<1.0.0 || >1.0.0"}},
		{"<1.0.0 || >=1.0.0", Constraint{[]versionRange{rangeAny}, "<1.0.0 || >=1.0.0"}}, // this can be canonicalized to "any" because pre-releases are not allowed in those
		{"<1.0.0 || >=1.0.0-alpha", Constraint{[]versionRange{rangeAny, {&Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, &Version{1, 0, 0, nil, nil, "1.0.0"}, true, false, ">=1.0.0-alpha <1.0.0"}}, "<1.0.0 || >=1.0.0-alpha"}},
		{"<1.0.0-pr.1 || >=1.0.0", Constraint{[]versionRange{rangeAny, {&Version{1, 0, 0, []string{"0"}, nil, "1.0.0-0"}, &Version{1, 0, 0, []string{"pr", "1"}, nil, "1.0.0-pr.1"}, true, false, ">=1.0.0-0 <1.0.0-pr.1"}}, "<1.0.0-pr.1 || >=1.0.0"}},
	}

	for _, test := range tests {
		actual, err := NewConstraint(test.constraint)
		if err != nil {
			t.Errorf("NewConstraint(%s) returned error %s", test.constraint, err)
		}
		if !reflect.DeepEqual(actual, test.expected) {
			t.Errorf("NewConstraint(%s) = %s, expected %s", test.constraint, actual, test.expected)
		}
	}
}
