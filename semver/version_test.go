package semver

import (
	"reflect"
	"testing"
)

func TestMakeVersion_Valid(t *testing.T) {
	type test struct {
		version  string
		expected Version
	}

	var tests = []test{
		{"1.2.3", Version{1, 2, 3, nil, nil, "1.2.3"}},
		{"1.2.3-alpha", Version{1, 2, 3, []string{"alpha"}, nil, "1.2.3-alpha"}},
		{"1.2.3+build", Version{1, 2, 3, nil, []string{"build"}, "1.2.3+build"}},
		{"1.2.3-alpha+build", Version{1, 2, 3, []string{"alpha"}, []string{"build"}, "1.2.3-alpha+build"}},
		{"v1", Version{1, 0, 0, nil, nil, "v1"}},
		{"1.0.0+build-test", Version{1, 0, 0, nil, []string{"build-test"}, "1.0.0+build-test"}},
	}

	for _, test := range tests {
		v, err := NewVersion(test.version)
		if err != nil {
			t.Errorf("error parsing version %s: %s", test.version, err)
		}
		if !reflect.DeepEqual(v, test.expected) {
			t.Errorf("expected %v, got %v", test.expected, v)
		}
	}
}

func TestMakeVersion_Invalid(t *testing.T) {
	type test struct {
		version string
	}

	var tests = []test{
		{"1.0.0.0"},
		{"-1"},
		{"1.0.0-"},
		{"1.0.0+"},
		{"1.0.0-+build"},
	}

	for _, test := range tests {
		_, err := NewVersion(test.version)
		if err == nil {
			t.Errorf("expected error parsing version %s, got nil", test.version)
		}
	}
}

func TestVersion_Compare(t *testing.T) {
	type test struct {
		v1       Version
		v2       Version
		expected int
	}
	var tests = []test{
		{Version{1, 0, 0, nil, nil, "1.0.0"}, Version{1, 0, 0, nil, nil, "1.0.0"}, 0},
		{Version{1, 0, 0, nil, nil, "1.0.0"}, Version{1, 0, 0, nil, []string{"build"}, "1.0.0+build"}, 0},
		{Version{1, 0, 0, nil, nil, "1.0.0"}, Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, 1},
		{Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, Version{1, 0, 0, nil, nil, "1.0.0"}, -1},
		{Version{1, 0, 0, []string{"0"}, nil, "1.0.0-0"}, Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, -1},
		{Version{1, 0, 0, []string{"alpha"}, nil, "1.0.0-alpha"}, Version{1, 0, 0, []string{"alpha", "0"}, nil, "1.0.0-alpha.0"}, -1},
	}

	for _, test := range tests {
		actual := test.v1.Compare(test.v2)
		if actual != test.expected {
			t.Errorf("comparing %v and %v, expected %d, got %d", test.v1, test.v2, test.expected, actual)
		}
	}
}
