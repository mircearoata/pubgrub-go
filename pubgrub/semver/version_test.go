package semver

import (
	"fmt"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestMakeVersion_Valid(t *testing.T) {
	tests := []struct {
		version  string
		expected Version
	}{
		{"1.2.3", Version{1, 2, 3, nil, nil, "1.2.3"}},
		{"1.2.3-alpha", Version{1, 2, 3, []string{"alpha"}, nil, "1.2.3-alpha"}},
		{"1.2.3+build", Version{1, 2, 3, nil, []string{"build"}, "1.2.3+build"}},
		{"1.2.3-alpha+build", Version{1, 2, 3, []string{"alpha"}, []string{"build"}, "1.2.3-alpha+build"}},
		{"v1", Version{1, 0, 0, nil, nil, "v1"}},
		{"1.0.0+build-test", Version{1, 0, 0, nil, []string{"build-test"}, "1.0.0+build-test"}},
	}

	for _, test := range tests {
		v, err := NewVersion(test.version)
		testza.AssertNoError(t, err, "NewVersion(%s)", test.version)
		testza.AssertEqual(t, test.expected, v, "NewVersion(%s)", test.version)
		testza.AssertEqual(t, test.version, v.RawString(), "NewVersion(%s).RawString()", test.version)
	}
}

func TestMakeVersion_Invalid(t *testing.T) {
	tests := []struct {
		version string
	}{
		{"1.0.0.0"},
		{"-1"},
		{"1.0.0-"},
		{"1.0.0+"},
		{"1.0.0-+build"},
	}

	for _, test := range tests {
		_, err := NewVersion(test.version)
		testza.AssertEqual(t, fmt.Sprintf("invalid version string: %s", test.version), err.Error(), "NewVersion(%s)", test.version)
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		v1       Version
		v2       Version
		expected int
	}{
		//// Release
		// Equal
		{Version{1, 0, 0, nil, nil, "1.0.0"}, Version{1, 0, 0, nil, nil, "1.0.0"}, 0},

		// Patch diff
		{Version{1, 0, 0, nil, nil, "1.0.0"}, Version{1, 0, 1, nil, nil, "1.0.1"}, -1},
		{Version{1, 0, 1, nil, nil, "1.0.1"}, Version{1, 0, 0, nil, nil, "1.0.0"}, 1},

		// Minor diff
		{Version{1, 0, 0, nil, nil, "1.1.0"}, Version{1, 1, 0, nil, nil, "1.1.0"}, -1},
		{Version{1, 1, 0, nil, nil, "1.1.0"}, Version{1, 0, 0, nil, nil, "1.1.0"}, 1},

		// Major diff
		{Version{1, 0, 0, nil, nil, "1.0.0"}, Version{2, 0, 0, nil, nil, "2.0.0"}, -1},
		{Version{2, 0, 0, nil, nil, "2.0.0"}, Version{1, 0, 0, nil, nil, "1.0.0"}, 1},

		// Build metadata
		{Version{1, 0, 0, nil, nil, "1.0.0"}, Version{1, 0, 0, nil, []string{"build"}, "1.0.0+build"}, 0},

		//// Pre-release
		// Release and pre-release
		{Version{1, 0, 0, nil, nil, "1.0.0"}, Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, 1},

		// Equal
		{Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, Version{1, 0, 0, nil, nil, "1.0.0"}, -1},

		// Number and number
		{Version{1, 0, 0, []string{"1"}, nil, "1.0.0-1"}, Version{1, 0, 0, []string{"2"}, nil, "1.0.0-2"}, -1},
		{Version{1, 0, 0, []string{"2"}, nil, "1.0.0-2"}, Version{1, 0, 0, []string{"1"}, nil, "1.0.0-1"}, 1},

		// Number before string
		{Version{1, 0, 0, []string{"0"}, nil, "1.0.0-0"}, Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, -1},
		{Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, Version{1, 0, 0, []string{"0"}, nil, "1.0.0-0"}, 1},

		// Different pre-release length
		{Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, Version{1, 0, 0, []string{"alpha", "0"}, nil, "1.0.0-alpha.0"}, -1},
	}

	for _, test := range tests {
		actual := test.v1.Compare(test.v2)
		testza.AssertEqual(t, test.expected, actual, "Compare(%s, %s)", test.v1, test.v2)
	}
}
