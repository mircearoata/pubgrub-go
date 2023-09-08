package semver

import (
	"github.com/MarvinJWendt/testza"
	"testing"
)

func TestMakeVersionRange(t *testing.T) {
	type test struct {
		versionRange string
		expected     versionRange
	}

	var tests = []test{
		{"1.2.3", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, lowerInclusive: true, upperInclusive: true, raw: "1.2.3"}},
		{"=1.2.3", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, lowerInclusive: true, upperInclusive: true, raw: "=1.2.3"}},
		{">=1.2.3", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, lowerInclusive: true, raw: ">=1.2.3"}},
		{">1.2.3", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, raw: ">1.2.3"}},
		{"<=1.2.3", versionRange{upperBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperInclusive: true, raw: "<=1.2.3"}},
		{"<1.2.3", versionRange{upperBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, raw: "<1.2.3"}},
		// This one is not supported yet
		//{"1.2.3 - 1.2.4", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperBound: &Version{1, 2, 4, nil, nil, "1.2.4"}, lowerInclusive: true, upperInclusive: true, raw: "1.2.3 - 1.2.4"}},
		{">=1.2.3 <1.2.4", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperBound: &Version{1, 2, 4, nil, nil, "1.2.4"}, lowerInclusive: true, raw: ">=1.2.3 <1.2.4"}},
		{"^1.2.3", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperBound: &Version{2, 0, 0, nil, nil, "2.0.0"}, lowerInclusive: true, upperInclusive: false, raw: "^1.2.3"}},
		{"^1.2", versionRange{lowerBound: &Version{1, 2, 0, nil, nil, "1.2"}, upperBound: &Version{2, 0, 0, nil, nil, "2.0.0"}, lowerInclusive: true, upperInclusive: false, raw: "^1.2"}},
		{"^1", versionRange{lowerBound: &Version{1, 0, 0, nil, nil, "1"}, upperBound: &Version{2, 0, 0, nil, nil, "2.0.0"}, lowerInclusive: true, upperInclusive: false, raw: "^1"}},
		{"^0.2.3", versionRange{lowerBound: &Version{0, 2, 3, nil, nil, "0.2.3"}, upperBound: &Version{0, 3, 0, nil, nil, "0.3.0"}, lowerInclusive: true, upperInclusive: false, raw: "^0.2.3"}},
		{"^0.0.3", versionRange{lowerBound: &Version{0, 0, 3, nil, nil, "0.0.3"}, upperBound: &Version{0, 0, 4, nil, nil, "0.0.4"}, lowerInclusive: true, upperInclusive: false, raw: "^0.0.3"}},
		{"^1.0.0-1", versionRange{lowerBound: &Version{1, 0, 0, []string{"1"}, nil, "1.0.0-1"}, upperBound: &Version{2, 0, 0, nil, nil, "2.0.0"}, lowerInclusive: true, upperInclusive: false, raw: "^1.0.0-1"}},
		{"~1.0.0", versionRange{lowerBound: &Version{1, 0, 0, nil, nil, "1.0.0"}, upperBound: &Version{1, 1, 0, nil, nil, "1.1.0"}, lowerInclusive: true, upperInclusive: false, raw: "~1.0.0"}},
		{"~1.0", versionRange{lowerBound: &Version{1, 0, 0, nil, nil, "1.0"}, upperBound: &Version{1, 1, 0, nil, nil, "1.1.0"}, lowerInclusive: true, upperInclusive: false, raw: "~1.0"}},
		{"~1", versionRange{lowerBound: &Version{1, 0, 0, nil, nil, "1"}, upperBound: &Version{2, 0, 0, nil, nil, "2.0.0"}, lowerInclusive: true, upperInclusive: false, raw: "~1"}},

		{"*", versionRange{raw: "*"}},
	}

	for _, test := range tests {
		actual, err := makeVersionRange(test.versionRange)
		testza.AssertNoError(t, err, "makeVersionRange(%s)", test.versionRange)
		testza.AssertEqual(t, test.expected, actual, "makeVersionRange(%s)", test.versionRange)
	}
}

func TestVersionRange_Contains(t *testing.T) {
	type test struct {
		versionRange string
		version      string
		expected     bool
	}

	var tests = []test{
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "1.2.4", false},
		{"1.2.3", "1.2.2", false},
		{"1.2.3", "1.2.3-alpha", false},
		{"1.2.3", "1.2.3+build", true},
		{"1.2.3", "1.2.3-alpha+build", false},

		{"=1.2.3", "1.2.3", true},

		{">=1.2.3", "1.2.3", true},
		{">=1.2.3", "1.2.4", true},
		{">=1.2.3", "2.0.0", true},
		{">=1.2.3", "1.2.2", false},
		{">=1.2.3", "1.2.3-alpha", false},

		{">1.2.3", "1.2.3", false},
		{">1.2.3", "1.2.4", true},

		{"<=1.2.3", "1.2.3", true},
		{"<=1.2.3", "1.2.2", true},
		{"<=1.2.3", "1.2.3-0", false},

		{"<1.2.3", "1.2.3", false},
		{"<1.2.3", "1.2.2", true},

		{"^1.2.3", "1.2.3", true},
		{"^1.2.3", "1.2.4", true},
		{"^1.2.3", "1.3.0", true},
		{"^1.2.3", "2.0.0", false},
		{"^1.2.3", "1.2.2", false},
		{"^1.2.3", "1.2.3-alpha", false},

		{"^1.2", "1.2.3", true},
		{"^1.2", "1.3.0", true},
		{"^1.2", "2.0.0", false},

		{"^1", "1.2.3", true},
		{"^1", "2.0.0", false},

		{"^0.2.3", "0.2.3", true},
		{"^0.2.3", "0.2.4", true},
		{"^0.2.3", "0.3.0", false},

		{"^0.0.3", "0.0.3", true},
		{"^0.0.3", "0.0.4", false},

		{"^1.0.0-1", "1.0.0-1", true},
		{"^1.0.0-1", "1.0.0-2", true},
		{"^1.0.0-1", "1.0.0", true},
		{"^1.0.0-1", "1.0.0-0", false},
		{"^1.0.0-1", "1.0.0-alpha", true},
		{"^1.0.0-1", "1.0.1-1", false},

		{"~1.0.0", "1.0.0", true},
		{"~1.0.0", "1.0.1", true},
		{"~1.0.0", "1.1.0", false},

		{"~1", "1.0.0", true},
		{"~1", "1.0.1", true},
		{"~1", "1.1.0", true},
		{"~1", "2.0.0", false},
	}

	for _, test := range tests {
		vr, err := makeVersionRange(test.versionRange)
		testza.AssertNoError(t, err, "makeVersionRange(%s)", test.versionRange)
		v, err := NewVersion(test.version)
		testza.AssertNoError(t, err, "NewVersion(%s)", test.version)
		testza.AssertEqual(t, test.expected, vr.Contains(v), "%s contains %s", test.versionRange, test.version)
	}
}
