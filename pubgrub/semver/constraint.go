package semver

import (
	"slices"
	"strings"
)

var AnyConstraint = Constraint{
	ranges: []versionRange{
		rangeAny,
	},
}

type Constraint struct {
	ranges []versionRange

	raw string
}

func NewConstraint(c string) (Constraint, error) {
	ranges := strings.Split(c, "||")
	result := Constraint{
		raw: c,
	}
	for _, r := range ranges {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		vr, err := makeVersionRange(r)
		if err != nil {
			return Constraint{}, err
		}
		result.ranges = append(result.ranges, vr)
	}
	return result.canonical(), nil
}

func SingleVersionConstraint(v Version) Constraint {
	vRange := versionRange{
		raw: v.raw,
	}
	vRange = vRange.withLowerBound(v, true)
	vRange = vRange.withUpperBound(v, true)
	return makeCanonicalConstraint([]versionRange{vRange}, v.raw)
}

// NewConstraintFromVersionSubset returns a minimal constraint that matches exactly the given versions out of the
// given set of all versions. Both slices must be sorted in ascending order
func NewConstraintFromVersionSubset(versions []Version, allVersions []Version) Constraint {
	var ranges []versionRange
	i := 0
	for _, v := range versions {
		for ; i < len(allVersions); i++ {
			if allVersions[i].Compare(v) == 0 {
				break
			}
		}
		if i == len(allVersions) {
			panic("should be unreachable")
		}

		if i == 0 {
			ranges = append(ranges, versionRange{
				upperBound:     &allVersions[i],
				upperInclusive: true,
			})
		}
		if i < len(allVersions)-1 {
			ranges = append(ranges, versionRange{
				lowerBound:     &allVersions[i],
				lowerInclusive: true,
				upperBound:     &allVersions[i+1],
				upperInclusive: false,
			})
		} else {
			ranges = append(ranges, versionRange{
				lowerBound:     &allVersions[i],
				lowerInclusive: true,
			})
		}
	}

	c := makeCanonicalConstraint(ranges, "")
	c.raw = c.String()
	return c
}

func makeCanonicalConstraint(ranges []versionRange, raw string) Constraint {
	return Constraint{
		ranges: ranges,
		raw:    raw,
	}.canonical()
}

// canonical returns a new Constraint that is equivalent to v
// but which contains no two overlapping ranges, and which
// is sorted in ascending order of the lower bound of each range.
func (v Constraint) canonical() Constraint {
	type versionOnAxis struct {
		version     *Version
		isInclusive bool

		isUpper bool
	}

	versions := make([]versionOnAxis, 0, len(v.ranges)*2)
	for _, r := range v.ranges {
		versions = append(versions, versionOnAxis{
			version:     r.lowerBound,
			isInclusive: r.lowerInclusive,
			isUpper:     false,
		})
		versions = append(versions, versionOnAxis{
			version:     r.upperBound,
			isInclusive: r.upperInclusive,
			isUpper:     true,
		})
	}

	slices.SortFunc(versions, func(a, b versionOnAxis) int {
		if a.version != nil && b.version != nil {
			result := a.version.Compare(*b.version)
			if result != 0 {
				return result
			}
			// If the versions are equal, order the lower bound before the upper bound for the merge to continue,
			// but only if the one of the bounds is inclusive
			if a.isUpper != b.isUpper && (a.isInclusive || b.isInclusive) {
				if a.isUpper {
					return 1
				}
				return -1
			}
			// If the versions are the same version and type, order the inclusive bound at the outer point based on type
			if a.isInclusive != b.isInclusive {
				if a.isUpper {
					if a.isInclusive {
						return 1
					}
					return -1
				}
				if a.isInclusive {
					return -1
				}
				return 1
			}

			// everything is equal
			return 0
		}
		if a.version == nil && b.version == nil {
			if a.isUpper != b.isUpper {
				if a.isUpper {
					return 1
				}
				return -1
			}
			return 0
		}
		if a.version == nil {
			if a.isUpper {
				return 1
			}
			return -1
		}
		if b.version == nil {
			if b.isUpper {
				return -1
			}
			return 1
		}
		return 0
	})

	result := Constraint{
		raw: v.raw,
	}

	nestedCount := 0
	var currentRange versionRange
	for i := 0; i < len(versions); i++ {
		if versions[i].isUpper {
			nestedCount--
		} else {
			nestedCount++
			if nestedCount == 1 {
				currentRange = versionRange{
					lowerBound:     versions[i].version,
					lowerInclusive: versions[i].isInclusive,
				}
			}
		}
		if nestedCount == 0 {
			currentRange.upperBound = versions[i].version
			currentRange.upperInclusive = versions[i].isInclusive
			currentRange.raw = currentRange.String()

			result.ranges = append(result.ranges, currentRange)
		}
	}

	// At this point no two ranges are overlapping, therefore no two ranges have an equal lower bound
	slices.SortFunc(result.ranges, func(a, b versionRange) int {
		lowerA := &Version{0, 0, 0, nil, nil, ""}
		lowerB := &Version{0, 0, 0, nil, nil, ""}
		if a.lowerBound != nil {
			lowerA = a.lowerBound
		}
		if b.lowerBound != nil {
			lowerB = b.lowerBound
		}

		return lowerA.Compare(*lowerB)
	})

	return result
}

func (v Constraint) IsEmpty() bool {
	return len(v.ranges) == 0
}

func (v Constraint) Intersect(other Constraint) Constraint {
	if v.IsEmpty() || other.IsEmpty() {
		return Constraint{}
	}
	var ranges []versionRange
	for _, r := range v.ranges {
		for _, r2 := range other.ranges {
			intersection := r.Intersect(r2)
			if !intersection.IsEmpty() {
				ranges = append(ranges, intersection)
			}
		}
	}
	c := makeCanonicalConstraint(ranges, "")
	c.raw = c.String()
	return c
}

func (v Constraint) Union(other Constraint) Constraint {
	return makeCanonicalConstraint(append(v.ranges, other.ranges...), v.raw+" || "+other.raw)
}

func (v Constraint) Difference(other Constraint) Constraint {
	return v.Intersect(other.Inverse())
}

func (v Constraint) Contains(other Version) bool {
	for _, r := range v.ranges {
		if r.Contains(other) {
			return true
		}
	}
	return false
}

func (v Constraint) Inverse() Constraint {
	result := AnyConstraint
	for _, r := range v.ranges {
		result = result.Intersect(r.Inverse())
	}
	return result.canonical()
}

func (v Constraint) IsAny() bool {
	return len(v.ranges) == 1 && v.ranges[0].Equal(rangeAny)
}

func (v Constraint) String() string {
	rangeStrings := make([]string, 0, len(v.ranges))
	for _, r := range v.ranges {
		rangeStrings = append(rangeStrings, r.String())
	}
	return strings.Join(rangeStrings, " || ")
}

func (v Constraint) RawString() string {
	return v.raw
}

func (v Constraint) Equal(other Constraint) bool {
	return slices.EqualFunc(v.ranges, other.ranges, func(a, b versionRange) bool {
		return a.Equal(b)
	})
}
