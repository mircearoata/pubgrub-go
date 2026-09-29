package pubgrub

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mircearoata/pubgrub-go/pubgrub/semver"
)

type StandardCauseStrings struct {
	Because    string
	AndBecause string
	SoBecause  string
	Thus       string
	AndCauses  string
	CauseRef   string
}

var DefaultCauseStrings = StandardCauseStrings{
	Because:    "Because %s, %s.",
	AndBecause: "And because %s, %s.",
	SoBecause:  "So, because %s, %s.",
	Thus:       "Thus, %s.",
	AndCauses:  "%s and %s",
	CauseRef:   "%s (%d)",
}

type StandardIncompatibilityStrings struct {
	ResolvingFailed string

	DependsOn       string
	Installing      string
	Forbids         string
	IsForbidden     string
	AreIncompatible string
	IsRequired      string
	ListSeparator   string
}

var DefaultIncompatibilityStrings = StandardIncompatibilityStrings{
	ResolvingFailed: "version solving failed",

	DependsOn:       "%s depends on %s",
	Installing:      "installing %s",
	Forbids:         "%s forbids %s",
	IsForbidden:     "%s is forbidden",
	AreIncompatible: "%s are incompatible",
	IsRequired:      "%s is required",
	ListSeparator:   ", ",
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
	if len(terms) == 1 {
		t := terms[0]
		if t.Positive() {
			return fmt.Sprintf(w.strings.IsForbidden, termStringer.Term(t, true))
		}
		return fmt.Sprintf(w.strings.IsRequired, termStringer.Term(t, false))
	}
	if len(terms) >= 3 {
		slices.SortFunc(terms, func(a, b Term) int {
			return strings.Compare(a.Dependency(), b.Dependency())
		})
		formattedTerms := make([]string, 0, len(terms))
		for _, t := range terms {
			formattedTerms = append(formattedTerms, termStringer.Term(t, true))
		}
		return fmt.Sprintf(w.strings.AreIncompatible, strings.Join(formattedTerms, w.strings.ListSeparator))
	}
	var pkg, dep Term
	if terms[0].Positive() {
		pkg = terms[0]
		dep = terms[1]
	} else {
		pkg = terms[1]
		dep = terms[0]
	}
	if dep.Positive() {
		if c.dependant != "" {
			// This is an optional dependency, which has a positive term, but with an inverse constraint
			// We revert the constraint here to get the term in a similar format to the others
			if pkg.Dependency() != c.dependant {
				pkg, dep = dep, pkg
			}
		} else {
			// What can we do here to determine a logical order of the terms?
			// For now, we can just order them by the package name,
			// so that the order is consistent between runs at least

			// Maybe we can do some heuristics on the version constraint
			// to see for which of the terms the inverse makes more sense than the original
			// One such heuristic could be the number of ranges in the constraint

			if pkg.Dependency() > dep.Dependency() {
				pkg, dep = dep, pkg
			}
		}
		dep = dep.Inverse()
	}
	if pkg.Dependency() == rootPkg {
		return fmt.Sprintf(w.strings.Installing, termStringer.Term(dep, false))
	}
	if dep.Constraint().IsEmpty() {
		return fmt.Sprintf(w.strings.Forbids, termStringer.Term(pkg, true), termStringer.Term(dep, false))
	}
	return fmt.Sprintf(w.strings.DependsOn, termStringer.Term(pkg, true), termStringer.Term(dep, false))
}
