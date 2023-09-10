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
		//// Single end simple
		{"1.2.3", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, lowerInclusive: true, upperInclusive: true, raw: "1.2.3"}},
		{"=1.2.3", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, lowerInclusive: true, upperInclusive: true, raw: "=1.2.3"}},
		{">=1.2.3", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, lowerInclusive: true, raw: ">=1.2.3"}},
		{">1.2.3", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, raw: ">1.2.3"}},
		{"<=1.2.3", versionRange{upperBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperInclusive: true, raw: "<=1.2.3"}},
		{"<1.2.3", versionRange{upperBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, raw: "<1.2.3"}},

		//// Double end simple
		// This one is not supported yet
		//{"1.2.3 - 1.2.4", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperBound: &Version{1, 2, 4, nil, nil, "1.2.4"}, lowerInclusive: true, upperInclusive: true, raw: "1.2.3 - 1.2.4"}},
		{">=1.2.3 <1.2.4", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperBound: &Version{1, 2, 4, nil, nil, "1.2.4"}, lowerInclusive: true, raw: ">=1.2.3 <1.2.4"}},

		//// Double end complex
		{">=1.0.0 >1.0.1 <=2.0.0 <1.9.9", versionRange{&Version{1, 0, 1, nil, nil, "1.0.1"}, &Version{1, 9, 9, nil, nil, "1.9.9"}, false, false, ">=1.0.0 >1.0.1 <=2.0.0 <1.9.9"}},
		{"^2.3.4 <2.5.0", versionRange{&Version{2, 3, 4, nil, nil, "2.3.4"}, &Version{2, 5, 0, nil, nil, "2.5.0"}, true, false, "^2.3.4 <2.5.0"}},

		//// Double end syntax sugar
		{"^1.2.3", versionRange{lowerBound: &Version{1, 2, 3, nil, nil, "1.2.3"}, upperBound: &Version{2, 0, 0, nil, nil, "2.0.0"}, lowerInclusive: true, upperInclusive: false, raw: "^1.2.3"}},
		{"^1.2", versionRange{lowerBound: &Version{1, 2, 0, nil, nil, "1.2"}, upperBound: &Version{2, 0, 0, nil, nil, "2.0.0"}, lowerInclusive: true, upperInclusive: false, raw: "^1.2"}},
		{"^1", versionRange{lowerBound: &Version{1, 0, 0, nil, nil, "1"}, upperBound: &Version{2, 0, 0, nil, nil, "2.0.0"}, lowerInclusive: true, upperInclusive: false, raw: "^1"}},
		{"^0.2.3", versionRange{lowerBound: &Version{0, 2, 3, nil, nil, "0.2.3"}, upperBound: &Version{0, 3, 0, nil, nil, "0.3.0"}, lowerInclusive: true, upperInclusive: false, raw: "^0.2.3"}},
		{"^0.0.3", versionRange{lowerBound: &Version{0, 0, 3, nil, nil, "0.0.3"}, upperBound: &Version{0, 0, 4, nil, nil, "0.0.4"}, lowerInclusive: true, upperInclusive: false, raw: "^0.0.3"}},
		{"^1.0.0-1", versionRange{lowerBound: &Version{1, 0, 0, []string{"1"}, nil, "1.0.0-1"}, upperBound: &Version{2, 0, 0, nil, nil, "2.0.0"}, lowerInclusive: true, upperInclusive: false, raw: "^1.0.0-1"}},
		{"~1.0.0", versionRange{lowerBound: &Version{1, 0, 0, nil, nil, "1.0.0"}, upperBound: &Version{1, 1, 0, nil, nil, "1.1.0"}, lowerInclusive: true, upperInclusive: false, raw: "~1.0.0"}},
		{"~1.0", versionRange{lowerBound: &Version{1, 0, 0, nil, nil, "1.0"}, upperBound: &Version{1, 1, 0, nil, nil, "1.1.0"}, lowerInclusive: true, upperInclusive: false, raw: "~1.0"}},
		{"~1", versionRange{lowerBound: &Version{1, 0, 0, nil, nil, "1"}, upperBound: &Version{2, 0, 0, nil, nil, "2.0.0"}, lowerInclusive: true, upperInclusive: false, raw: "~1"}},

		//// X ranges
		// Currently unsupported, "*" is specifically handled
		{"*", versionRange{raw: "*"}},
	}

	for _, test := range tests {
		actual, err := makeVersionRange(test.versionRange)
		testza.AssertNoError(t, err, "makeVersionRange(%s)", test.versionRange)
		testza.AssertEqual(t, test.expected, actual, "makeVersionRange(%s)", test.versionRange)
		testza.AssertEqual(t, test.versionRange, actual.RawString(), "makeVersionRange(%s).RawString()", test.versionRange)
	}
}

func TestMakeVersionRange_Invalid(t *testing.T) {
	type test struct {
		versionRange string
	}

	var tests = []test{
		{">=a"},
		{"<=a"},
		{"^a"},
		{"~a"},
		{"a"},
	}

	for _, test := range tests {
		_, err := makeVersionRange(test.versionRange)
		testza.AssertNotEqual(t, nil, err, "makeVersionRange(%s)", test.versionRange)
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

func TestVersionRange_IsEmpty(t *testing.T) {
	type test struct {
		versionRange string
		expected     bool
	}

	var tests = []test{
		{"1.2.3", false},
		{"=1.2.3", false},
		{">=1.2.3", false},
		{">1.2.3", false},
		{"<=1.2.3", false},
		{"<1.2.3", false},
		{"^1.2.3", false},
		{"^1.2", false},
		{"^1", false},
		{"^0.2.3", false},
		{"^0.0.3", false},
		{"^1.0.0-1", false},
		{"~1.0.0", false},
		{"~1", false},
		{"*", false},

		{">1.0.0 <1.0.0", true},
		{"^1.0.0 ^2.0.0", true},
		{"^1.5.0 <01.2.3", true},
	}

	for _, test := range tests {
		vr, err := makeVersionRange(test.versionRange)
		testza.AssertNoError(t, err, "makeVersionRange(%s)", test.versionRange)
		testza.AssertEqual(t, test.expected, vr.IsEmpty(), "%s.IsEmpty()", test.versionRange)
	}
}

func TestVersionRange_Equal(t *testing.T) {
	type test struct {
		r1       string
		r2       string
		expected bool
	}

	var tests = []test{
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "1.2.4", false},
		{"1.2.3", "1.2.2", false},
		{"1.2.3", "1.2.3-alpha", false},
		{"1.2.3", "1.2.3+build", true},
		{"1.2.3", "1.2.3-alpha+build", false},

		{">=1.2.3", ">1.2.3", false},
		{"<=1.2.3", "<1.2.3", false},

		{">=1.2.3", ">=1.0.0 >=1.2.3", true},
		{">1.2.2", ">=1.2.3", false},
		{"<=1.2.3", "<=1.2.4", false},

		{"^1.2.3", ">=1.2.3 <2.0.0", true},
		{"^0.1.2", ">=0.1.2 <0.2.0", true},

		{">1.2.2", ">1.2.2 <1.2.3", false},
		{"<1.2.3", ">1.2.2 <1.2.3", false},
	}

	for _, test := range tests {
		r1, err := makeVersionRange(test.r1)
		testza.AssertNoError(t, err, "makeVersionRange(%s)", test.r1)
		r2, err := makeVersionRange(test.r2)
		testza.AssertNoError(t, err, "makeVersionRange(%s)", test.r2)
		testza.AssertEqual(t, test.expected, r1.Equal(r2), "%s.Equal(%s)", test.r1, test.r2)
	}
}

func TestVersionRange_Intersect(t *testing.T) {
	type test struct {
		r1       string
		r2       string
		expected versionRange
	}

	var tests = []test{
		//// Releases
		{"1.2.3", "1.2.3", versionRange{&Version{1, 2, 3, nil, nil, "1.2.3"}, &Version{1, 2, 3, nil, nil, "1.2.3"}, true, true, "1.2.3 1.2.3"}},
		{"1.2.3", "1.2.4", versionRange{&Version{1, 2, 4, nil, nil, "1.2.4"}, &Version{1, 2, 3, nil, nil, "1.2.3"}, true, true, "1.2.3 1.2.4"}},

		{">=1.2.3", "<1.5.0", versionRange{&Version{1, 2, 3, nil, nil, "1.2.3"}, &Version{1, 5, 0, nil, nil, "1.5.0"}, true, false, ">=1.2.3 <1.5.0"}},
		{">=1.2.3", "<=1.5.0", versionRange{&Version{1, 2, 3, nil, nil, "1.2.3"}, &Version{1, 5, 0, nil, nil, "1.5.0"}, true, true, ">=1.2.3 <=1.5.0"}},
		{"<=1.2.3", "<1.5.0", versionRange{nil, &Version{1, 2, 3, nil, nil, "1.2.3"}, false, true, "<=1.2.3 <1.5.0"}},
		{"<=1.2.3", "<=1.5.0", versionRange{nil, &Version{1, 2, 3, nil, nil, "1.2.3"}, false, true, "<=1.2.3 <=1.5.0"}},

		{">=1.0.0", "<=1.2.3", versionRange{&Version{1, 0, 0, nil, nil, "1.0.0"}, &Version{1, 2, 3, nil, nil, "1.2.3"}, true, true, ">=1.0.0 <=1.2.3"}},

		{"^1.2.3", "<1.9.9", versionRange{&Version{1, 2, 3, nil, nil, "1.2.3"}, &Version{1, 9, 9, nil, nil, "1.9.9"}, true, false, "^1.2.3 <1.9.9"}},

		//// Pre-releases
		{">=1.2.3-alpha", "<1.5.0", versionRange{&Version{1, 2, 3, []string{"alpha"}, nil, "1.2.3-alpha"}, &Version{1, 5, 0, nil, nil, "1.5.0"}, true, false, ">=1.2.3-alpha <1.5.0"}},
		{">=1.2.3-alpha", ">=1.0.0 <1.5.0", versionRange{&Version{1, 2, 3, []string{"alpha"}, nil, "1.2.3-alpha"}, &Version{1, 5, 0, nil, nil, "1.5.0"}, true, false, ">=1.2.3-alpha >=1.0.0 <1.5.0"}},
	}

	for _, test := range tests {
		r1, err := makeVersionRange(test.r1)
		testza.AssertNoError(t, err, "makeVersionRange(%s)", test.r1)
		r2, err := makeVersionRange(test.r2)
		testza.AssertNoError(t, err, "makeVersionRange(%s)", test.r2)
		testza.AssertEqual(t, test.expected, r1.Intersect(r2), "%s.Intersect(%s)", test.r1, test.r2)
	}
}
