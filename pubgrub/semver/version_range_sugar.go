package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	xVersion         = `([0-9]+|x|X|\*)(?:\.([0-9]+|x|X|\*))?(?:\.([0-9]+|x|X|\*))?(?:-([0-9A-Za-z\-]+(?:\.[0-9A-Za-z\-]+)*))?(?:\+([0-9A-Za-z\-]+(?:\.[0-9A-Za-z\-]+)*))?`
	xVersionNoGroups = strings.ReplaceAll(strings.ReplaceAll(xVersion, "(", "(?:"), "?:?:", "?:")
	hyphenRange      = regexp.MustCompile(fmt.Sprintf(`(%s) - (%s)`, xVersionNoGroups, xVersionNoGroups))
	caretRangeRegex  = regexp.MustCompile(fmt.Sprintf(`^\^%s$`, xVersion))
	tildeRangeRegex  = regexp.MustCompile(fmt.Sprintf(`^~%s$`, xVersion))
	xRangeRegex      = regexp.MustCompile(fmt.Sprintf(`^([<>]?=?)%s$`, xVersion))
)

func deSugarRange(v string) (string, error) {
	var err error
	v = replaceHyphens(v)
	v = replaceCarets(v)
	v = replaceTildes(v)
	v, err = replaceXRanges(v)
	if err != nil {
		return "", err
	}
	v = replaceStars(v)

	// Remove double spaces
	return strings.ReplaceAll(v, "  ", " "), nil
}

func replaceHyphens(v string) string {
	return hyphenRange.ReplaceAllString(v, ">=$1 <=$2")
}

func replaceCarets(v string) string {
	sections := strings.Split(v, " ")
	result := make([]string, 0, len(sections))
	for _, s := range sections {
		if s == "" {
			continue
		}
		result = append(result, replaceCaret(s))
	}
	return strings.Join(result, " ")
}

func replaceCaret(v string) string {
	match := caretRangeRegex.FindStringSubmatch(v)
	if len(match) == 0 {
		return v
	}
	versionString := v[1:] // Skip the ^
	majorString := match[1]
	minorString := match[2]
	patchString := match[3]

	xMajor := isX(majorString)
	xMinor := isX(minorString)
	xPatch := isX(patchString)

	// Might not parse because it's x, but that's checked later
	major, _ := strconv.Atoi(majorString)
	minor, _ := strconv.Atoi(minorString)
	patch, _ := strconv.Atoi(patchString)

	if xMajor {
		return "*"
	}
	if xMinor {
		return fmt.Sprintf(">=%d.0.0 <%d.0.0", major, major+1)
	}
	if xPatch {
		if major == 0 {
			return fmt.Sprintf(">=%d.%d.0 <%d.%d.0", major, minor, major, minor+1)
		}
		return fmt.Sprintf(">=%d.%d.0 <%d.0.0", major, minor, major+1)
	}
	if major != 0 {
		return fmt.Sprintf(">=%s <%d.%d.%d", versionString, major+1, 0, 0)
	}
	if minor != 0 {
		return fmt.Sprintf(">=%s <%d.%d.%d", versionString, major, minor+1, 0)
	}
	return fmt.Sprintf(">=%s <%d.%d.%d", versionString, major, minor, patch+1)
}

func replaceTildes(v string) string {
	sections := strings.Split(v, " ")
	result := make([]string, 0, len(sections))
	for _, s := range sections {
		if s == "" {
			continue
		}
		result = append(result, replaceTilde(s))
	}
	return strings.Join(result, " ")
}

func replaceTilde(v string) string {
	match := tildeRangeRegex.FindStringSubmatch(v)
	if len(match) == 0 {
		return v
	}
	versionString := v[1:] // Skip the ~
	majorString := match[1]
	minorString := match[2]
	patchString := match[3]

	xMajor := isX(majorString)
	xMinor := isX(minorString)
	xPatch := isX(patchString)

	// Might not parse because it's x, but that's checked later
	major, _ := strconv.Atoi(majorString)
	minor, _ := strconv.Atoi(minorString)

	if xMajor {
		return "*"
	}
	if xMinor {
		return fmt.Sprintf(">=%d.0.0 <%d.0.0", major, major+1)
	}
	if xPatch {
		// ~1.2 == >=1.2.0 <1.3.0
		return fmt.Sprintf(">=%d.%d.0 <%d.%d.0", major, minor, major, minor+1)
	}
	return fmt.Sprintf(">=%s <%d.%d.%d", versionString, major, minor+1, 0)
}

func replaceXRanges(v string) (string, error) {
	sections := strings.Split(v, " ")
	result := make([]string, 0, len(sections))
	for _, s := range sections {
		if s == "" {
			continue
		}
		s, err := replaceXRange(s)
		if err != nil {
			return "", err
		}
		result = append(result, s)
	}
	return strings.Join(result, " "), nil
}

func replaceXRange(v string) (string, error) {
	match := xRangeRegex.FindStringSubmatch(v)
	if len(match) == 0 {
		// At this point all other range sugar should have been replaced
		return "", fmt.Errorf("invalid comparator string: %s", v)
	}

	comparator := match[1]
	majorString := match[2]
	minorString := match[3]
	patchString := match[4]

	xMajor := isX(majorString)
	xMinor := isX(minorString)
	xPatch := isX(patchString)

	// Check that we have at least one x
	if !xMajor && !xMinor && !xPatch {
		return v, nil
	}

	if comparator == "=" {
		comparator = ""
	}

	if xMajor {
		if comparator == ">" || comparator == "<" {
			// Nothing can be allowed
			return "<0.0.0", nil
		}
		// Everything is allowed
		return "*", nil
	}

	// Must parse because it's not x
	major, _ := strconv.Atoi(majorString)
	// Might not parse because it's x, but that's ok
	minor, _ := strconv.Atoi(minorString)
	// We don't need the patch, since either minor or patch is x, therefore the patch is overwritten

	switch comparator {
	case ">":
		// >1 => >=2.0.0
		if xMinor {
			return fmt.Sprintf(">=%d.0.0", major+1), nil
		}
		// >1.2 => >=1.3.0
		return fmt.Sprintf(">=%d.%d.0", major, minor+1), nil
	case ">=":
		// >=1 => >=1.0.0
		if xMinor {
			return fmt.Sprintf(">=%d.0.0", major), nil
		}
		// >=1.2 => >=1.2.0
		return fmt.Sprintf(">=%d.%d.0", major, minor), nil
	case "<":
		// <1 => <1.0.0
		if xMinor {
			return fmt.Sprintf("<%d.0.0", major), nil
		}
		// <1.2 => <1.2.0
		return fmt.Sprintf("<%d.%d.0", major, minor), nil
	case "<=":
		// <=1 => <2.0.0
		if xMinor {
			return fmt.Sprintf("<%d.0.0", major+1), nil
		}
		// <=1.2 => <1.3.0
		return fmt.Sprintf("<%d.%d.0", major, minor+1), nil
	default:
		// =1 => >=1.0.0 <2.0.0
		if xMinor {
			return fmt.Sprintf(">=%d.0.0 <%d.0.0", major, major+1), nil
		}
		// =1.2 => >=1.2.0 <1.3.0
		return fmt.Sprintf(">=%d.%d.0 <%d.%d.0", major, minor, major, minor+1), nil
	}
}

func isX(v string) bool {
	return v == "x" || v == "X" || v == "*" || v == ""
}

func replaceStars(v string) string {
	return strings.ReplaceAll(v, "*", "")
}
