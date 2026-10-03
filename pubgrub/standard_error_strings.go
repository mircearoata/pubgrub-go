package pubgrub

import (
	"fmt"
	"strings"

	"github.com/mircearoata/pubgrub-go/pubgrub/semver"
)

type StandardCauseStrings struct {
	Because    string
	AndBecause string
	SoBecause  string
	Thus       string

	AndCauses string
	CauseRef  string

	DependsOnBoth  string
	RequiresBoth   string
	InstallingBoth string

	WhichDependsOn string
	WhichRequires  string

	WhichIsForbidden       string
	WhichNoVersions        string
	WhichNotFound          string
	WhichIsForbiddenReason string

	Alternative string
}

var DefaultCauseStrings = StandardCauseStrings{
	Because:    "Because %s, %s.",
	AndBecause: "And because %s, %s.",
	SoBecause:  "So, because %s, %s.",
	Thus:       "Thus, %s.",

	AndCauses: "%s and %s",
	CauseRef:  "%s (%d)",

	DependsOnBoth:  "%s depends on both %s and %s",
	RequiresBoth:   "%s requires both %s and %s",
	InstallingBoth: "installing both %s and %s",

	WhichDependsOn: "%s which depends on %s",
	WhichRequires:  "%s which requires %s",

	WhichIsForbidden:       "%s which is forbidden",
	WhichNoVersions:        "%s which matches no versions",
	WhichNotFound:          "%s which could not be found",
	WhichIsForbiddenReason: "%s which %s",

	Alternative: " or ",
}

type StandardIncompatibilityStrings struct {
	ResolvingFailed string

	DependsOn         string
	OptionalDependsOn string
	Requires          string
	Installing        string
	Forbids           string
	IsForbidden       string
	IsRequired        string

	NoVersions        string
	NotFound          string
	IsForbiddenReason string
	IsInstalled       string

	IncompatibleWith string
	Either           string
	RequiresOneOf    string
	IfThen           string
	OneMustBeFalse   string
	OneMustBeTrue    string

	Alternative string
	Conjunction string
}

var DefaultIncompatibilityStrings = StandardIncompatibilityStrings{
	ResolvingFailed: "version solving failed",

	DependsOn:         "%s depends on %s",
	OptionalDependsOn: "%s optionally depends on %s",
	Requires:          "%s requires %s",
	Installing:        "installing %s",
	Forbids:           "%s forbids %s",
	IsForbidden:       "%s is forbidden",
	IsRequired:        "%s is required",

	NoVersions:        "%s matches no versions",
	NotFound:          "%s could not be found",
	IsForbiddenReason: "%s %s",
	IsInstalled:       "%s is installed",

	IncompatibleWith: "%s is incompatible with %s",
	Either:           "either %s or %s",
	RequiresOneOf:    "%s requires %s",
	IfThen:           "if %s then %s",
	OneMustBeFalse:   "one of %s must be false",
	OneMustBeTrue:    "one of %s must be true",

	Alternative: " or ",
	Conjunction: " and ",
}

type StandardTermStrings struct {
	EveryVersionOf string
	Default        string
}

var DefaultTermStrings = StandardTermStrings{
	EveryVersionOf: "every version of %s",
	Default:        "%s \"%s\"",
}

type IncompatibilityStringer interface {
	IncompatibilityString(incompatibility *Incompatibility, ts TermStringer, rootPkg string) string
}

type TwoCausesConfig interface {
	AllowMerging(c1 ReportCause, c2 ReportCause, rootPkg string) bool
}

type TermStringer interface {
	Term(pkg string, constraint semver.Constraint, allowEvery bool) string
}

type PackageFormatter interface {
	FormatPackage(pkg string) string
}

type ConstraintFormatter interface {
	FormatConstraint(pkg string, c semver.Constraint) string
}

type StandardTermStringer struct {
	strings             StandardTermStrings
	packageFormatter    PackageFormatter
	constraintFormatter ConstraintFormatter
}

func NewStandardTermStringer() StandardTermStringer {
	return StandardTermStringer{strings: DefaultTermStrings}
}

func (w StandardTermStringer) WithStrings(strings StandardTermStrings) StandardTermStringer {
	w.strings = strings
	return w
}

func (w StandardTermStringer) WithPackageFormatter(f PackageFormatter) StandardTermStringer {
	w.packageFormatter = f
	return w
}

func (w StandardTermStringer) WithConstraintFormatter(f ConstraintFormatter) StandardTermStringer {
	w.constraintFormatter = f
	return w
}

func (w StandardTermStringer) FormatPackage(pkg string) string {
	if w.packageFormatter != nil {
		return w.packageFormatter.FormatPackage(pkg)
	}
	return pkg
}

func (w StandardTermStringer) FormatConstraint(pkg string, c semver.Constraint) string {
	if w.constraintFormatter != nil {
		return w.constraintFormatter.FormatConstraint(pkg, c)
	}
	return c.String()
}

func (w StandardTermStringer) Term(pkg string, constraint semver.Constraint, allowEvery bool) string {
	pkgName := w.FormatPackage(pkg)
	if constraint.IsAny() {
		if allowEvery {
			return fmt.Sprintf(w.strings.EveryVersionOf, pkgName)
		}
		return pkgName
	}
	if constraint.IsEmpty() {
		return pkgName
	}
	constraintStr := w.FormatConstraint(pkg, constraint)
	return fmt.Sprintf(w.strings.Default, pkgName, constraintStr)
}

type StandardIncompatibilityStringer struct {
	strings StandardIncompatibilityStrings
}

func NewStandardIncompatibilityStringer() StandardIncompatibilityStringer {
	return StandardIncompatibilityStringer{strings: DefaultIncompatibilityStrings}
}

func (w StandardIncompatibilityStringer) WithStrings(strings StandardIncompatibilityStrings) StandardIncompatibilityStringer {
	w.strings = strings
	return w
}

func (w StandardIncompatibilityStringer) IncompatibilityString(c *Incompatibility, termStringer TermStringer, rootPkg string) string {
	cause := c.Cause()
	switch typedCause := cause.(type) {
	case RootCause:
		// This should never be reached, as packages should not depend on the root package, but just in case
		return fmt.Sprintf(w.strings.IsRequired, termStringer.Term(rootPkg, semver.AnyConstraint, false))
	case DependencyCause:
		if typedCause.Pkg == rootPkg {
			return fmt.Sprintf(w.strings.Installing, termStringer.Term(typedCause.Target, typedCause.Constraint, false))
		}
		if typedCause.Constraint.IsEmpty() {
			return fmt.Sprintf(w.strings.Forbids, termStringer.Term(typedCause.Pkg, typedCause.PkgRange, true), termStringer.Term(typedCause.Target, typedCause.Constraint, false))
		}
		if typedCause.Optional {
			return fmt.Sprintf(w.strings.OptionalDependsOn, termStringer.Term(typedCause.Pkg, typedCause.PkgRange, true), termStringer.Term(typedCause.Target, typedCause.Constraint, false))
		}
		return fmt.Sprintf(w.strings.DependsOn, termStringer.Term(typedCause.Pkg, typedCause.PkgRange, true), termStringer.Term(typedCause.Target, typedCause.Constraint, false))
	case NoVersionsCause:
		return fmt.Sprintf(w.strings.NoVersions, termStringer.Term(typedCause.Pkg, typedCause.Constraint, false))
	case PackageNotFoundCause:
		return fmt.Sprintf(w.strings.NotFound, termStringer.Term(typedCause.Pkg, semver.AnyConstraint, false))
	case PackageVersionForbiddenCause:
		return fmt.Sprintf(w.strings.IsForbiddenReason, termStringer.Term(typedCause.Pkg, typedCause.PkgRange, true), typedCause.Reason)
	case EnvironmentPackageCause:
		return fmt.Sprintf(w.strings.IsInstalled, termStringer.Term(typedCause.Pkg, typedCause.Constraint, false))
	}

	terms := c.Terms()
	positives := make([]Term, 0, len(terms))
	negatives := make([]Term, 0, len(terms))
	for _, t := range terms {
		if t.Positive() {
			positives = append(positives, t)
		} else {
			negatives = append(negatives, t)
		}
	}

	if len(terms) == 0 || (len(terms) == 1 && len(positives) == 1 && positives[0].pkg == rootPkg) {
		return w.strings.ResolvingFailed
	}

	if len(terms) == 1 {
		if len(positives) == 1 {
			return fmt.Sprintf(w.strings.IsForbidden, FormatTerm(positives[0], termStringer, true))
		}
		return fmt.Sprintf(w.strings.IsRequired, FormatTerm(negatives[0], termStringer, false))
	}

	if len(terms) == 2 {
		switch {
		case len(positives) == 1:
			pkg, dep := positives[0], negatives[0]
			if dep.Constraint().IsEmpty() {
				return fmt.Sprintf(w.strings.Forbids, FormatTerm(pkg, termStringer, true), FormatTerm(dep, termStringer, false))
			}
			return fmt.Sprintf(w.strings.Requires, FormatTerm(pkg, termStringer, true), FormatTerm(dep, termStringer, false))

		case len(negatives) == 0:
			return fmt.Sprintf(w.strings.IncompatibleWith,
				FormatTerm(positives[0], termStringer, true),
				FormatTerm(positives[1], termStringer, false))

		default:
			return fmt.Sprintf(w.strings.Either,
				FormatTerm(negatives[0], termStringer, false),
				FormatTerm(negatives[1], termStringer, false))
		}
	}

	switch {
	case len(positives) == 1:
		return fmt.Sprintf(w.strings.RequiresOneOf,
			FormatTerm(positives[0], termStringer, true),
			w.joinTerms(negatives, false, w.strings.Alternative, termStringer))
	case len(negatives) >= 1 && len(positives) > 1:
		return fmt.Sprintf(w.strings.IfThen,
			w.joinTerms(positives, false, w.strings.Conjunction, termStringer),
			w.joinTerms(negatives, false, w.strings.Alternative, termStringer))
	case len(positives) > 0:
		return fmt.Sprintf(w.strings.OneMustBeFalse, w.joinTerms(positives, false, w.strings.Alternative, termStringer))
	}

	return fmt.Sprintf(w.strings.OneMustBeTrue, w.joinTerms(negatives, false, w.strings.Alternative, termStringer))
}

func FormatTerm(term Term, ts TermStringer, allowEvery bool) string {
	return ts.Term(term.pkg, term.versionConstraint, allowEvery)
}

func FormatTerms(terms []Term, ts TermStringer, allowEvery bool) []string {
	res := make([]string, len(terms))
	for i, t := range terms {
		res[i] = FormatTerm(t, ts, allowEvery)
	}
	return res
}

func (w StandardIncompatibilityStringer) joinTerms(terms []Term, allowEvery bool, separator string, termStringer TermStringer) string {
	return strings.Join(FormatTerms(terms, termStringer, allowEvery), separator)
}
