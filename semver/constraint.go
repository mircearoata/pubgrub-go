package semver

import (
	"fmt"
	"math"
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
		result.ranges = append(result.ranges, *vr)
	}
	return result.canonical(), nil
}

func SingleVersionConstraint(v Version) Constraint {
	vRange := versionRange{
		raw: v.raw,
	}
	vRange = vRange.withLowerBound(&v, true)
	vRange = vRange.withUpperBound(&v, true)
	return makeConstraint([]versionRange{vRange}, v.raw)
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

	c := makeConstraint(ranges, "")
	c.raw = c.String()
	return c
}

func makeConstraint(ranges []versionRange, raw string) Constraint {
	return Constraint{
		ranges: ranges,
		raw:    raw,
	}.canonical()
}

// canonical returns a new Constraint that is equivalent to v
// but which contains no two overlapping ranges, and which
// is sorted in ascending order of the lower bound of each range.
func (v Constraint) canonical() Constraint {
	var ranges []versionRange
	for _, r := range v.ranges {
		if r.IsEmpty() {
			// empty ranges are ignored since they would never match anything
			continue
		}

		// handle ranges endpoints that mention pre-releases separately since they have extra meaning
		// split them into two ranges, one that only matches pre-releases and one that only matches releases

		// also check if both ends of the range are pre-releases of the same version, in which case we can just use the original range
		if r.lowerBound != nil && r.upperBound != nil &&
			r.lowerBound.isPrerelease() && r.upperBound.isPrerelease() &&
			r.lowerBound.isSameRelease(*r.upperBound) {
			ranges = append(ranges, r)
			continue
		}

		newRange := r
		if r.lowerBound != nil && r.lowerBound.isPrerelease() {
			nextPatch := r.lowerBound.nextPatch()
			prereleaseRange := versionRange{
				lowerBound:     r.lowerBound,
				lowerInclusive: r.lowerInclusive,
				upperBound:     &nextPatch,
				upperInclusive: false,
			}
			prereleaseRange.raw = prereleaseRange.String()
			ranges = append(ranges, prereleaseRange)
			newRange.lowerBound = &nextPatch
			newRange.lowerInclusive = true
		}

		if r.upperBound != nil && r.upperBound.isPrerelease() {
			prereleaseStart := Version{
				major: r.upperBound.major,
				minor: r.upperBound.minor,
				patch: r.upperBound.patch,
				pre:   []string{"0"},
				raw:   fmt.Sprintf("%d.%d.%d-0", r.upperBound.major, r.upperBound.minor, r.upperBound.patch),
			}
			prereleaseRange := versionRange{
				lowerBound:     &prereleaseStart,
				lowerInclusive: true,
				upperBound:     r.upperBound,
				upperInclusive: r.upperInclusive,
			}
			prereleaseRange.raw = prereleaseRange.String()
			ranges = append(ranges, prereleaseRange)
			// Since <x.y.z-0 is equivalent to <x.y.z, we use the latter instead which results in more ranges being merged
			nextPatch := prereleaseStart.nextPatch()
			newRange.upperBound = &nextPatch
			newRange.upperInclusive = false
		}
		ranges = append(ranges, newRange)
	}

	type versionOnAxis struct {
		version     *Version
		isInclusive bool

		isUpper          bool
		isFromPrerelease bool
	}

	var versions []versionOnAxis
	for _, r := range ranges {
		isFromPrerelease := false
		if r.lowerBound != nil && r.lowerBound.isPrerelease() {
			isFromPrerelease = true
		}
		if r.upperBound != nil && r.upperBound.isPrerelease() {
			isFromPrerelease = true
		}
		versions = append(versions, versionOnAxis{
			version:          r.lowerBound,
			isInclusive:      r.lowerInclusive,
			isUpper:          false,
			isFromPrerelease: isFromPrerelease,
		})
		versions = append(versions, versionOnAxis{
			version:          r.upperBound,
			isInclusive:      r.upperInclusive,
			isUpper:          true,
			isFromPrerelease: isFromPrerelease,
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
				} else {
					return -1
				}
			}
			// If the versions are the same version and type, order the inclusive bound at the outer point based on type
			if a.isInclusive != b.isInclusive {
				if a.isUpper {
					if a.isInclusive {
						return 1
					} else {
						return -1
					}
				} else {
					if a.isInclusive {
						return -1
					} else {
						return 1
					}
				}
			}

			// everything is equal
			return 0
		}
		if a.version == nil && b.version == nil {
			if a.isUpper != b.isUpper {
				if a.isUpper {
					return 1
				} else {
					return -1
				}
			}
			return 0
		}
		if a.version == nil {
			if a.isUpper {
				return 1
			} else {
				return -1
			}
		}
		if b.version == nil {
			if b.isUpper {
				return -1
			} else {
				return 1
			}
		}
		panic("unreachable")
	})

	result := Constraint{
		raw: v.raw,
	}

	nestedCount := 0
	var currentRange *versionRange
	nestedPrereleaseCount := 0
	var currentPrereleaseRange *versionRange
	for i := 0; i < len(versions); i++ {
		nested := &nestedCount
		curRange := &currentRange
		if versions[i].isFromPrerelease {
			nested = &nestedPrereleaseCount
			curRange = &currentPrereleaseRange
		}

		if versions[i].isUpper {
			*nested--
		} else {
			*nested++
			if *nested == 1 {
				*curRange = &versionRange{
					lowerBound:     versions[i].version,
					lowerInclusive: versions[i].isInclusive,
				}
			}
		}
		if *nested == 0 {
			(*curRange).upperBound = versions[i].version
			(*curRange).upperInclusive = versions[i].isInclusive
			(*curRange).raw = (*curRange).String()

			result.ranges = append(result.ranges, **curRange)
			curRange = nil
		}
	}

	slices.SortFunc(result.ranges, func(a, b versionRange) int {
		lowerA := &Version{0, 0, 0, nil, nil, ""}
		lowerB := &Version{0, 0, 0, nil, nil, ""}
		if a.lowerBound != nil {
			lowerA = a.lowerBound
		}
		if b.lowerBound != nil {
			lowerB = b.lowerBound
		}

		if res := lowerA.Compare(*lowerB); res != 0 {
			return res
		}

		upperA := &Version{math.MaxInt32, math.MaxInt32, math.MaxInt32, nil, nil, ""}
		upperB := &Version{math.MaxInt32, math.MaxInt32, math.MaxInt32, nil, nil, ""}
		if a.upperBound != nil {
			upperA = a.upperBound
		}
		if b.upperBound != nil {
			upperB = b.upperBound
		}

		return upperA.Compare(*upperB)
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
	raw := ""
	for _, r := range v.ranges {
		for _, r2 := range other.ranges {
			intersection := r.Intersect(r2)
			if !intersection.IsEmpty() {
				ranges = append(ranges, intersection)
				raw += intersection.raw + " || "
			}
		}
	}
	raw = strings.TrimSuffix(raw, " || ")
	return makeConstraint(ranges, raw)
}

func (v Constraint) Union(other Constraint) Constraint {
	return makeConstraint(append(v.ranges, other.ranges...), v.raw+" || "+other.raw)
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
	if v.IsEmpty() {
		return AnyConstraint
	}
	result := v.ranges[0].Inverse()
	for _, r := range v.ranges {
		result = result.Intersect(r.Inverse())
	}
	return result.canonical()
}

func (v Constraint) IsAny() bool {
	return len(v.ranges) == 1 && v.ranges[0].Equal(rangeAny)
}

func (v Constraint) String() string {
	var rangeStrings []string
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
