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
	InstallingBoth string

	WhichDependsOn   string
	WhichIsForbidden string
	Alternative      string
}

var DefaultCauseStrings = StandardCauseStrings{
	Because:    "Because %s, %s.",
	AndBecause: "And because %s, %s.",
	SoBecause:  "So, because %s, %s.",
	Thus:       "Thus, %s.",

	AndCauses: "%s and %s",
	CauseRef:  "%s (%d)",

	DependsOnBoth:  "%s depends on both %s and %s",
	InstallingBoth: "installing both %s and %s",

	WhichDependsOn:   "%s which depends on %s",
	WhichIsForbidden: "%s which is forbidden",
	Alternative:      " or ",
}

type StandardIncompatibilityStrings struct {
	ResolvingFailed string

	DependsOn   string
	Installing  string
	Forbids     string
	IsForbidden string
	IsRequired  string

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

	DependsOn:   "%s depends on %s",
	Installing:  "installing %s",
	Forbids:     "%s forbids %s",
	IsForbidden: "%s is forbidden",
	IsRequired:  "%s is required",

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

type TermStringer interface {
	Term(t Term, allowEvery bool) string
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

func (w StandardTermStringer) Term(t Term, allowEvery bool) string {
	pkgName := w.FormatPackage(t.Dependency())
	if t.Constraint().IsAny() {
		if allowEvery {
			return fmt.Sprintf(w.strings.EveryVersionOf, pkgName)
		}
		return pkgName
	}
	if t.Constraint().IsEmpty() {
		return pkgName
	}
	constraintStr := w.FormatConstraint(t.Dependency(), t.Constraint())
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

func (w StandardIncompatibilityStringer) isRoot(incompatibility *Incompatibility, rootPkg string) bool {
	terms := incompatibility.Terms()
	return len(terms) == 0 || (len(terms) == 1 && terms[0].Positive() && terms[0].Dependency() == rootPkg)
}

func (w StandardIncompatibilityStringer) IncompatibilityString(c *Incompatibility, termStringer TermStringer, rootPkg string) string {
	if w.isRoot(c, rootPkg) {
		return w.strings.ResolvingFailed
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

	if len(terms) == 1 {
		if len(positives) == 1 {
			return fmt.Sprintf(w.strings.IsForbidden, termStringer.Term(positives[0], true))
		}
		return fmt.Sprintf(w.strings.IsRequired, termStringer.Term(negatives[0], false))
	}

	if len(terms) == 2 {
		switch {
		case len(positives) == 1:
			pkg, dep := positives[0], negatives[0]
			if pkg.Dependency() == rootPkg {
				return fmt.Sprintf(w.strings.Installing, termStringer.Term(dep, false))
			}
			if dep.Constraint().IsEmpty() {
				return fmt.Sprintf(w.strings.Forbids, termStringer.Term(pkg, true), termStringer.Term(dep, false))
			}
			return fmt.Sprintf(w.strings.DependsOn, termStringer.Term(pkg, true), termStringer.Term(dep, false))

		case len(negatives) == 0:
			if c.dependant == "" {
				return fmt.Sprintf(w.strings.IncompatibleWith,
					termStringer.Term(positives[0], true),
					termStringer.Term(positives[1], false))
			}
			// This is an optional dependency, which has a positive term, but with an inverse constraint
			// We revert the constraint here to get the term in a similar format to the others
			pkg, dep := positives[0], positives[1]
			if pkg.Dependency() != c.dependant {
				pkg, dep = dep, pkg
			}
			dep = dep.Inverse()
			if dep.Constraint().IsEmpty() {
				return fmt.Sprintf(w.strings.Forbids, termStringer.Term(pkg, true), termStringer.Term(dep, false))
			}
			return fmt.Sprintf(w.strings.DependsOn, termStringer.Term(pkg, true), termStringer.Term(dep, false))

		default:
			return fmt.Sprintf(w.strings.Either,
				termStringer.Term(negatives[0], false),
				termStringer.Term(negatives[1], false))
		}
	}

	switch {
	case len(positives) == 1:
		return fmt.Sprintf(w.strings.RequiresOneOf,
			termStringer.Term(positives[0], true),
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

func FormatTerms(terms []Term, ts TermStringer, allowEvery bool) []string {
	res := make([]string, len(terms))
	for i, t := range terms {
		res[i] = ts.Term(t, allowEvery)
	}
	return res
}

func (w StandardIncompatibilityStringer) joinTerms(terms []Term, allowEvery bool, separator string, termStringer TermStringer) string {
	return strings.Join(FormatTerms(terms, termStringer, allowEvery), separator)
}
