package semver

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type Version struct {
	major int
	minor int
	patch int
	pre   []string
	build []string

	raw string
}

var semVerRegex = regexp.MustCompile(`^v?([0-9]+)(?:\.([0-9]+))?(?:\.([0-9]+))?(?:-([0-9A-Za-z\-]+(?:\.[0-9A-Za-z\-]+)*))?(?:\+([0-9A-Za-z\-]+(?:\.[0-9A-Za-z\-]+)*))?$`)

func NewVersion(v string) (Version, error) {
	match := semVerRegex.FindStringSubmatch(v)
	if len(match) == 0 {
		return Version{}, fmt.Errorf("invalid version string: %s", v)
	}
	result := Version{
		raw: v,
	}
	if match[1] != "" {
		result.major, _ = strconv.Atoi(match[1])
	}
	if match[2] != "" {
		result.minor, _ = strconv.Atoi(match[2])
	}
	if match[3] != "" {
		result.patch, _ = strconv.Atoi(match[3])
	}
	if match[4] != "" {
		result.pre = strings.Split(match[4], ".")
	}
	if match[5] != "" {
		result.build = strings.Split(match[5], ".")
	}
	return result, nil
}

func (v Version) Compare(other Version) int {
	if v.major != other.major {
		return v.major - other.major
	}
	if v.minor != other.minor {
		return v.minor - other.minor
	}
	if v.patch != other.patch {
		return v.patch - other.patch
	}
	if !v.IsPrerelease() && other.IsPrerelease() {
		return 1
	}
	if v.IsPrerelease() && !other.IsPrerelease() {
		return -1
	}
	// both are pre-releases or releases
	return slices.CompareFunc(v.pre, other.pre, func(a, b string) int {
		aNum, aErr := strconv.Atoi(a)
		bNum, bErr := strconv.Atoi(b)
		if aErr == nil && bErr == nil {
			// both are numbers
			return aNum - bNum
		}
		if aErr != nil && bErr != nil {
			// both are strings
			return strings.Compare(a, b)
		}
		// numbers go before strings
		if aErr != nil {
			// a is a string, b is a number
			return 1
		}
		// a is a number, b is a string
		return -1
	})
}

func (v Version) Inverse() Constraint {
	return makeCanonicalConstraint([]versionRange{
		{
			upperBound:     &v,
			upperInclusive: false,
		},
		{
			lowerBound:     &v,
			lowerInclusive: false,
		},
	}, fmt.Sprintf("<%s || >%s", v.raw, v.raw))
}

func (v Version) bumpPatch() Version {
	return Version{
		major: v.major,
		minor: v.minor,
		patch: v.patch + 1,
		raw:   fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch+1),
	}
}

func (v Version) bumpMinor() Version {
	return Version{
		major: v.major,
		minor: v.minor + 1,
		patch: 0,
		raw:   fmt.Sprintf("%d.%d.0", v.major, v.minor+1),
	}
}

func (v Version) bumpMajor() Version {
	return Version{
		major: v.major + 1,
		minor: 0,
		patch: 0,
		raw:   fmt.Sprintf("%d.0.0", v.major+1),
	}
}

func (v Version) IsPrerelease() bool {
	return len(v.pre) != 0
}

func (v Version) String() string {
	result := fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
	if len(v.pre) != 0 {
		result += "-" + strings.Join(v.pre, ".")
	}
	if len(v.build) != 0 {
		result += "+" + strings.Join(v.build, ".")
	}
	return result
}

func (v Version) RawString() string {
	return v.raw
}
