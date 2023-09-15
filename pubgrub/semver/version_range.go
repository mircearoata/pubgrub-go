package semver

import (
	"strings"

	"github.com/pkg/errors"
)

var rangeAny = versionRange{raw: "*"}

// versionRange represents a continuous range of versions, including pre-releases
// We do not care about the individual versions and symbols mentioned, but rather compute the
// resulting intersection of all the ranges that the version should match and only store its endpoints.
type versionRange struct {
	lowerBound     *Version
	upperBound     *Version
	lowerInclusive bool
	upperInclusive bool

	raw string
}

func makeVersionRange(v string) (versionRange, error) {
	result := versionRange{
		raw: v,
	}
	v, err := deSugarRange(v)
	if err != nil {
		return versionRange{}, errors.Wrapf(err, "failed to de-sugar range %s", v)
	}
	sections := strings.Split(v, " ")
	for _, s := range sections {
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
			switch {
			case result < 0:
				newLowerBound = other.lowerBound
				newLowerInclusive = other.lowerInclusive
			case result > 0:
				newLowerBound = v.lowerBound
				newLowerInclusive = v.lowerInclusive
			default:
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
			switch {
			case result < 0:
				newUpperBound = v.upperBound
				newUpperInclusive = v.upperInclusive
			case result > 0:
				newUpperBound = other.upperBound
				newUpperInclusive = other.upperInclusive
			default:
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
		// If x.y.z is considered incompatible (<x.y.z),
		// then pre-releases of that version are also likely considered incompatible
		// Therefore, we treat <x.y.z as <x.y.z-0
		var upperBound Version
		if !v.upperInclusive && !v.upperBound.IsPrerelease() {
			upperBound = *v.upperBound
			upperBound.pre = []string{"0"}
		} else {
			upperBound = *v.upperBound
		}

		result := upperBound.Compare(other)
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

	return makeCanonicalConstraint(ranges, raw)
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
			switch {
			case v.lowerBound.major != 0:
				nextCaretVersion = v.lowerBound.bumpMajor()
			case v.lowerBound.minor != 0:
				nextCaretVersion = v.lowerBound.bumpMinor()
			default:
				nextCaretVersion = v.lowerBound.bumpPatch()
			}
			if v.upperBound.Compare(nextCaretVersion) == 0 {
				return "^" + v.lowerBound.String()
			}

			// Shorthand for tilde version
			var nextTildeVersion Version
			switch {
			case v.lowerBound.minor != 0:
				nextTildeVersion = v.lowerBound.bumpMinor()
			default:
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
