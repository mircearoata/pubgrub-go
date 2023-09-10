package semver

import (
	"github.com/pkg/errors"
	"strings"
)

var rangeAny = versionRange{raw: "*"}

// versionRange represents a range of versions as defined by https://www.npmjs.com/package/semver#range-grammar
// We do not care about the individual versions and symbols mentioned, but rather compute the
// resulting intersection of all the ranges that the version should match and only store its endpoints.
// TODO: currently we do not support parsing the hyphen range and the x-range
type versionRange struct {
	lowerBound     *Version
	upperBound     *Version
	lowerInclusive bool
	upperInclusive bool

	raw string
}

func makeVersionRange(v string) (versionRange, error) {
	sections := strings.Split(v, " ")
	result := versionRange{
		raw: v,
	}
	for _, s := range sections {
		if s == "*" {
			// The default range is any
			continue
		}
		if s == "" {
			continue
		}
		if s[0] == '>' {
			// lower bound
			inclusive := false
			versionString := s[1:]
			if s[1] == '=' {
				inclusive = true
				versionString = s[2:]
			}
			ver, err := NewVersion(versionString)
			if err != nil {
				return versionRange{}, errors.Wrapf(err, "invalid version string parsing primitive range %s", s)
			}
			result = result.withLowerBound(ver, inclusive)
			continue
		}
		if s[0] == '<' {
			// upper bound
			inclusive := false
			versionString := s[1:]
			if s[1] == '=' {
				inclusive = true
				versionString = s[2:]
			}
			ver, err := NewVersion(versionString)
			if err != nil {
				return versionRange{}, errors.Wrapf(err, "invalid version string parsing primitive range %s", s)
			}
			result = result.withUpperBound(ver, inclusive)
			continue
		}
		if s[0] == '^' {
			// caret range
			lowerBound, err := NewVersion(strings.TrimPrefix(s, "^"))
			if err != nil {
				return versionRange{}, errors.Wrapf(err, "invalid version string parsing caret range %s", s)
			}
			result = result.withLowerBound(lowerBound, true)
			var upperBound Version
			if lowerBound.major != 0 {
				upperBound = lowerBound.bumpMajor()
			} else if lowerBound.minor != 0 {
				upperBound = lowerBound.bumpMinor()
			} else {
				upperBound = lowerBound.bumpPatch()
			}
			result = result.withUpperBound(upperBound, false)
			continue
		}
		if s[0] == '~' {
			// tilde range
			lowerBound, err := NewVersion(strings.TrimPrefix(s, "~"))
			if err != nil {
				return versionRange{}, errors.Wrapf(err, "invalid version string parsing caret range %s", s)
			}
			result = result.withLowerBound(lowerBound, true)
			var upperBound Version
			if strings.Contains(s, ".") {
				// minor version specified, allow only patch updates
				upperBound = lowerBound.bumpMinor()
			} else {
				// only major version specified, allow minor and patch updates
				upperBound = lowerBound.bumpMajor()
			}
			result = result.withUpperBound(upperBound, false)
			continue
		}
		// exact version
		ver, err := NewVersion(strings.TrimPrefix(s, "="))
		if err != nil {
			return versionRange{}, errors.Wrapf(err, "invalid version string parsing primitive range %s", s)
		}
		result = result.withLowerBound(ver, true)
		result = result.withUpperBound(ver, true)
	}
	return result, nil
}

func (v versionRange) withLowerBound(ver Version, inclusive bool) versionRange {
	if v.lowerBound == nil {
		v.lowerBound = &ver
		v.lowerInclusive = inclusive
	} else {
		cmp := ver.Compare(*v.lowerBound)
		if cmp > 0 {
			v.lowerBound = &ver
			v.lowerInclusive = inclusive
		} else if cmp == 0 {
			v.lowerInclusive = v.lowerInclusive && inclusive
		}
	}
	return v
}

func (v versionRange) withUpperBound(ver Version, inclusive bool) versionRange {
	if v.upperBound == nil {
		v.upperBound = &ver
		v.upperInclusive = inclusive
	} else {
		cmp := ver.Compare(*v.upperBound)
		if cmp < 0 {
			v.upperBound = &ver
			v.upperInclusive = inclusive
		} else if cmp == 0 {
			v.upperInclusive = v.upperInclusive && inclusive
		}
	}
	return v
}

func (v versionRange) IsEmpty() bool {
	if v.lowerBound != nil && v.upperBound != nil {
		result := v.lowerBound.Compare(*v.upperBound)
		if result > 0 {
			// lower bound is greater than upper bound
			return true
		} else if result == 0 && (!v.lowerInclusive || !v.upperInclusive) {
			// lower bound is equal to upper bound, but one of them is not inclusive
			return true
		}
	}
	return false
}

func (v versionRange) Intersect(other versionRange) versionRange {
	var newLowerBound *Version
	var newLowerInclusive bool
	if v.lowerBound == nil {
		newLowerBound = other.lowerBound
		newLowerInclusive = other.lowerInclusive
	} else {
		if other.lowerBound == nil {
			newLowerBound = v.lowerBound
			newLowerInclusive = v.lowerInclusive
		} else {
			result := v.lowerBound.Compare(*other.lowerBound)
			if result < 0 {
				newLowerBound = other.lowerBound
				newLowerInclusive = other.lowerInclusive
			} else if result > 0 {
				newLowerBound = v.lowerBound
				newLowerInclusive = v.lowerInclusive
			} else {
				newLowerBound = v.lowerBound
				newLowerInclusive = v.lowerInclusive && other.lowerInclusive
			}
		}
	}

	var newUpperBound *Version
	var newUpperInclusive bool
	if v.upperBound == nil {
		newUpperBound = other.upperBound
		newUpperInclusive = other.upperInclusive
	} else {
		if other.upperBound == nil {
			newUpperBound = v.upperBound
			newUpperInclusive = v.upperInclusive
		} else {
			result := v.upperBound.Compare(*other.upperBound)
			if result < 0 {
				newUpperBound = v.upperBound
				newUpperInclusive = v.upperInclusive
			} else if result > 0 {
				newUpperBound = other.upperBound
				newUpperInclusive = other.upperInclusive
			} else {
				newUpperBound = v.upperBound
				newUpperInclusive = v.upperInclusive && other.upperInclusive
			}
		}
	}

	return versionRange{
		lowerBound:     newLowerBound,
		upperBound:     newUpperBound,
		lowerInclusive: newLowerInclusive,
		upperInclusive: newUpperInclusive,
		raw:            strings.TrimSpace(v.raw + " " + other.raw),
	}
}

func (v versionRange) Contains(other Version) bool {
	if v.lowerBound != nil {
		result := v.lowerBound.Compare(other)
		if v.lowerInclusive {
			if result > 0 {
				// lower bound is greater than other
				return false
			}
		} else {
			if result >= 0 {
				// lower bound is greater than or equal to other
				return false
			}
		}
	}
	if v.upperBound != nil {
		result := v.upperBound.Compare(other)
		if v.upperInclusive {
			if result < 0 {
				// upper bound is less than other
				return false
			}
		} else {
			if result <= 0 {
				// upper bound is less than or equal to other
				return false
			}
		}
	}
	if other.isPrerelease() {
		// pre-releases are only contained in ranges that have the same major, minor, and patch as one of the endpoints
		// and that endpoint also has a pre-release
		// this second part is not technically required for the lower bound, since pre-releases are ordered before releases,
		// but it is required for the upper bound
		matchesEnd := false
		if v.lowerBound != nil && v.lowerBound.isPrerelease() {
			if other.major == v.lowerBound.major && other.minor == v.lowerBound.minor && other.patch == v.lowerBound.patch {
				matchesEnd = true
			}
		}
		if v.upperBound != nil && v.upperBound.isPrerelease() {
			if other.major == v.upperBound.major && other.minor == v.upperBound.minor && other.patch == v.upperBound.patch {
				matchesEnd = true
			}
		}
		if !matchesEnd {
			return false
		}
	}
	return true
}

func (v versionRange) Inverse() Constraint {
	if v.upperBound == nil && v.lowerBound == nil {
		return Constraint{}
	}
	var ranges []versionRange
	raw := ""
	if v.lowerBound != nil {
		ranges = append(ranges, versionRange{
			upperBound:     v.lowerBound,
			upperInclusive: !v.lowerInclusive,
		})
		if v.lowerInclusive {
			raw += "<"
		} else {
			raw += "<="
		}
		raw += v.lowerBound.raw
	}
	raw += " || "
	if v.upperBound != nil {
		ranges = append(ranges, versionRange{
			lowerBound:     v.upperBound,
			lowerInclusive: !v.upperInclusive,
		})
		if v.upperInclusive {
			raw += ">"
		} else {
			raw += ">="
		}
		raw += v.upperBound.raw
	}
	raw = strings.TrimPrefix(raw, " || ")
	raw = strings.TrimSuffix(raw, " || ")

	return makeConstraint(ranges, raw)
}

func (v versionRange) Equal(other versionRange) bool {
	if v.lowerBound != nil && other.lowerBound != nil {
		if v.lowerBound.Compare(*other.lowerBound) != 0 {
			return false
		}
		if v.lowerInclusive != other.lowerInclusive {
			return false
		}
	} else if v.lowerBound != nil || other.lowerBound != nil {
		return false
	}
	if v.upperBound != nil && other.upperBound != nil {
		if v.upperBound.Compare(*other.upperBound) != 0 {
			return false
		}
		if v.upperInclusive != other.upperInclusive {
			return false
		}
	} else if v.upperBound != nil || other.upperBound != nil {
		return false
	}
	return true
}

func (v versionRange) String() string {
	// Shorthand for any range
	if v.upperBound == nil && v.lowerBound == nil {
		return "*"
	}
	if v.upperBound != nil && v.lowerBound != nil && v.lowerInclusive {
		// Shorthand for exact version
		if v.upperBound.Compare(*v.lowerBound) == 0 {
			return v.lowerBound.String()
		}

		if !v.upperInclusive {
			// Shorthand for caret version
			var nextCaretVersion Version
			if v.lowerBound.major != 0 {
				nextCaretVersion = v.lowerBound.bumpMajor()
			} else if v.lowerBound.minor != 0 {
				nextCaretVersion = v.lowerBound.bumpMinor()
			} else if v.lowerBound.patch != 0 {
				nextCaretVersion = v.lowerBound.bumpPatch()
			}
			if v.upperBound.Compare(nextCaretVersion) == 0 {
				return "^" + v.lowerBound.String()
			}

			// Shorthand for tilde version
			var nextTildeVersion Version
			if v.lowerBound.minor != 0 {
				nextTildeVersion = v.lowerBound.bumpMinor()
			} else {
				nextTildeVersion = v.lowerBound.bumpMajor()
			}
			if v.upperBound.Compare(nextTildeVersion) == 0 {
				return "~" + v.lowerBound.String()
			}
		}
	}
	raw := ""
	if v.lowerBound != nil {
		if v.lowerInclusive {
			raw += ">="
		} else {
			raw += ">"
		}
		raw += v.lowerBound.String()
	}
	raw += " "
	if v.upperBound != nil {
		if v.upperInclusive {
			raw += "<="
		} else {
			raw += "<"
		}
		raw += v.upperBound.String()
	}
	return strings.TrimSpace(raw)
}

func (v versionRange) RawString() string {
	return v.raw
}
