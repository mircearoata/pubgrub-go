package pubgrub

import "fmt"

type StandardCauseStrings struct {
	TwoCauses             string
	TwoCausesFinal        string
	TwoCausesOneTag       string
	TwoCausesOneTagFinal  string
	TwoCausesTwoTags      string
	TwoCausesTwoTagsFinal string
	OneCause              string
	OneCauseFinal         string
	OneCauseOneTag        string
	OneCauseOneTagFinal   string
	NoCause               string
}

var DefaultCauseStrings = StandardCauseStrings{
	TwoCauses:             "Because %s and %s, %s.",
	TwoCausesFinal:        "So, because %s and %s, %s.",
	TwoCausesOneTag:       "Because %s and %s (%d), %s.",
	TwoCausesOneTagFinal:  "So, because %s and %s (%d), %s.",
	TwoCausesTwoTags:      "Because %s (%d) and %s (%d), %s.",
	TwoCausesTwoTagsFinal: "So, because %s (%d) and %s (%d), %s.",
	OneCause:              "And because %s, %s.",
	OneCauseFinal:         "So, because %s, %s.",
	OneCauseOneTag:        "And because %s (%d), %s.",
	OneCauseOneTagFinal:   "So, because %s (%d), %s.",
	NoCause:               "Thus, %s.",
}

type StandardIncompatibilityStrings struct {
	ResolvingFailed string

	DependsOn   string
	Installing  string
	Forbids     string
	IsForbidden string
}

var DefaultIncompatibilityStrings = StandardIncompatibilityStrings{
	ResolvingFailed: "version solving failed",

	DependsOn:   "%s depends on %s",
	Installing:  "installing %s",
	Forbids:     "%s forbids %s",
	IsForbidden: "%s is forbidden",
}

type StandardTermStringer struct{}

func (w StandardTermStringer) Term(t Term, includeVersion bool) string {
	if includeVersion {
		return t.String()
	}
	return t.Dependency()
}

type StandardIncompatibilityStringer struct {
	strings      StandardIncompatibilityStrings
	termStringer TermStringer
}

func NewStandardIncompatibilityStringer() StandardIncompatibilityStringer {
	return StandardIncompatibilityStringer{strings: DefaultIncompatibilityStrings, termStringer: StandardTermStringer{}}
}

func (w StandardIncompatibilityStringer) WithStrings(strings StandardIncompatibilityStrings) StandardIncompatibilityStringer {
	w.strings = strings
	return w
}

func (w StandardIncompatibilityStringer) WithTermStringer(termStringer TermStringer) StandardIncompatibilityStringer {
	w.termStringer = termStringer
	return w
}

func (w StandardIncompatibilityStringer) IsRoot(incompatibility *Incompatibility, rootPkg string) bool {
	terms := incompatibility.Terms()
	return len(terms) == 1 && terms[0].Positive() && terms[0].Dependency() == rootPkg
}

func (w StandardIncompatibilityStringer) IncompatibilityString(c *Incompatibility, rootPkg string) string {
	if w.IsRoot(c, rootPkg) {
		return w.strings.ResolvingFailed
	}
	terms := c.Terms()
	if len(terms) == 1 {
		t := terms[0]
		if t.Positive() {
			if t.Constraint().IsAny() {
				return fmt.Sprintf(w.strings.IsForbidden, w.termStringer.Term(t, false))
			}
			return fmt.Sprintf(w.strings.IsForbidden, w.termStringer.Term(t, true))
		}
		panic("negative term in cause")
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
		return fmt.Sprintf(w.strings.Installing, w.termStringer.Term(dep, true))
	}
	if dep.Constraint().IsEmpty() {
		return fmt.Sprintf(w.strings.Forbids, w.termStringer.Term(pkg, true), w.termStringer.Term(dep, false))
	}
	if dep.Constraint().IsAny() {
		return fmt.Sprintf(w.strings.DependsOn, w.termStringer.Term(pkg, true), w.termStringer.Term(dep, false))
	}
	return fmt.Sprintf(w.strings.DependsOn, w.termStringer.Term(pkg, true), w.termStringer.Term(dep, true))
}
