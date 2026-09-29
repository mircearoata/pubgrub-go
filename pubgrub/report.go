package pubgrub

import (
	"strings"
)

type LineKind uint8

const (
	LineSeparator LineKind = iota
	LineBothExternal
	LineBothReferenced
	LineReferencedAndExternal
	LinePriorAndExternal
	LineReferencedOnly
	LineExternalOnly
	LineNoCauses
)

type Cause struct {
	Incompatibility *Incompatibility
	Ref             int
}

type Line struct {
	Kind       LineKind
	Conclusion *Incompatibility
	Ref        int
	Final      bool
	Cause1     *Cause
	Cause2     *Cause
}

type Report struct {
	RootPackage string
	Lines       []Line
}

func NewReport(rootPkg string, cause *Incompatibility) *Report {
	report := &Report{RootPackage: rootPkg}
	if cause == nil {
		return report
	}
	b := &reportBuilder{
		rootPkg:     rootPkg,
		derivations: make(map[*Incompatibility]int),
		lineNumbers: make(map[*Incompatibility]int),
		nextLine:    1,
	}
	b.countDerivations(cause)
	b.visit(cause)
	report.Lines = b.lines
	return report
}

type reportBuilder struct {
	rootPkg     string
	derivations map[*Incompatibility]int
	lineNumbers map[*Incompatibility]int
	lines       []Line
	nextLine    int
}

func isDerived(c *Incompatibility) bool {
	return len(c.Causes()) == 2
}

func (r *reportBuilder) countDerivations(inc *Incompatibility) {
	if inc == nil {
		return
	}
	r.derivations[inc]++
	if r.derivations[inc] == 1 && isDerived(inc) {
		for _, cause := range inc.Causes() {
			r.countDerivations(cause)
		}
	}
}

func (r *reportBuilder) isSingleLine(inc *Incompatibility) bool {
	causes := inc.Causes()
	return len(causes) == 2 && !isDerived(causes[0]) && !isDerived(causes[1])
}

func (r *reportBuilder) isCollapsible(inc *Incompatibility) bool {
	if r.derivations[inc] > 1 {
		return false
	}
	causes := inc.Causes()
	if len(causes) != 2 {
		return false
	}
	c1, c2 := causes[0], causes[1]
	if isDerived(c1) == isDerived(c2) {
		return false
	}
	derived := c1
	if !isDerived(c1) {
		derived = c2
	}
	derivedCauses := derived.Causes()
	if len(derivedCauses) != 2 {
		return false
	}
	dc1, dc2 := derivedCauses[0], derivedCauses[1]
	var complexCause *Incompatibility
	if isDerived(dc1) {
		complexCause = dc1
	} else if isDerived(dc2) {
		complexCause = dc2
	}
	if complexCause != nil {
		if _, hasLine := r.lineNumbers[complexCause]; hasLine {
			return false
		}
	}
	return true
}

func (r *reportBuilder) orderCauses(c1, c2 *Incompatibility) (*Incompatibility, *Incompatibility) {
	// The negative term is the package that is depended on
	// Therefore we want the first incompatibility to be the one that has a negative term of the shared package
	for _, t1 := range c1.Terms() {
		for _, t2 := range c2.Terms() {
			if t1.Dependency() == t2.Dependency() {
				term1 := c1.get(t1.Dependency())
				if term1 != nil && !term1.Positive() {
					return c1, c2
				}
				term2 := c2.get(t2.Dependency())
				if term2 != nil && !term2.Positive() {
					return c2, c1
				}
			}
		}
	}
	// Otherwise just order by package name for consistency
	c1Pkgs := strings.Join(c1.Packages(), ",")
	c2Pkgs := strings.Join(c2.Packages(), ",")
	if c1Pkgs > c2Pkgs {
		return c2, c1
	}
	return c1, c2
}

func (r *reportBuilder) tagLastLine(inc *Incompatibility) {
	if len(r.lines) == 0 {
		return
	}

	num := r.nextLine
	r.nextLine++
	r.lineNumbers[inc] = num
	r.lines[len(r.lines)-1].Ref = num
}

func (r *reportBuilder) isRoot(incompatibility *Incompatibility) bool {
	terms := incompatibility.Terms()
	return len(terms) == 1 && terms[0].Positive() && terms[0].Dependency() == r.rootPkg
}

func (r *reportBuilder) visit(inc *Incompatibility) {
	if !isDerived(inc) {
		return
	}
	c1 := inc.Causes()[0]
	c2 := inc.Causes()[1]

	if isDerived(c1) && isDerived(c2) {
		l1, ok1 := r.lineNumbers[c1]
		l2, ok2 := r.lineNumbers[c2]

		if ok1 && ok2 {
			first, second := r.orderCauses(c1, c2)
			line1 := r.lineNumbers[first]
			line2 := r.lineNumbers[second]
			r.lines = append(r.lines, Line{
				Kind:       LineBothReferenced,
				Conclusion: inc,
				Final:      r.isRoot(inc),
				Cause1:     &Cause{Incompatibility: first, Ref: line1},
				Cause2:     &Cause{Incompatibility: second, Ref: line2},
			})
			return
		}

		if ok1 || ok2 {
			var withLine, withoutLine *Incompatibility
			var line int
			if ok1 {
				withLine, withoutLine, line = c1, c2, l1
			} else {
				withLine, withoutLine, line = c2, c1, l2
			}
			r.visit(withoutLine)
			r.lines = append(r.lines, Line{
				Kind:       LineReferencedOnly,
				Conclusion: inc,
				Final:      r.isRoot(inc),
				Cause1:     &Cause{Incompatibility: withLine, Ref: line},
			})
			return
		}

		single1 := r.isSingleLine(c1)
		single2 := r.isSingleLine(c2)
		if single1 || single2 {
			var first, second *Incompatibility
			if single2 {
				first, second = c1, c2
			} else {
				first, second = c2, c1
			}
			r.visit(first)
			r.visit(second)
			r.lines = append(r.lines, Line{
				Kind:       LineNoCauses,
				Conclusion: inc,
				Final:      r.isRoot(inc),
			})
			return
		}

		first, second := r.orderCauses(c1, c2)
		r.visit(first)
		r.tagLastLine(first)
		r.lines = append(r.lines, Line{
			Kind: LineSeparator,
		})
		r.visit(second)
		r.tagLastLine(second)
		firstLine := r.lineNumbers[first]
		r.lines = append(r.lines, Line{
			Kind:       LineReferencedOnly,
			Conclusion: inc,
			Final:      r.isRoot(inc),
			Cause1:     &Cause{Incompatibility: first, Ref: firstLine},
		})
		return
	}

	if isDerived(c1) != isDerived(c2) {
		derived, external := c1, c2
		if !isDerived(c1) {
			derived, external = c2, c1
		}

		if derivedLine, ok := r.lineNumbers[derived]; ok {
			r.lines = append(r.lines, Line{
				Kind:       LineReferencedAndExternal,
				Conclusion: inc,
				Final:      r.isRoot(inc),
				Cause1:     &Cause{Incompatibility: external},
				Cause2:     &Cause{Incompatibility: derived, Ref: derivedLine},
			})
			return
		}

		if r.isCollapsible(derived) {
			derivedCauses := derived.Causes()
			dc1, dc2 := derivedCauses[0], derivedCauses[1]
			var priorDerived, priorExternal *Incompatibility
			if isDerived(dc1) {
				priorDerived, priorExternal = dc1, dc2
			} else {
				priorDerived, priorExternal = dc2, dc1
			}
			r.visit(priorDerived)
			r.lines = append(r.lines, Line{
				Kind:       LinePriorAndExternal,
				Conclusion: inc,
				Final:      r.isRoot(inc),
				Cause1:     &Cause{Incompatibility: priorExternal},
				Cause2:     &Cause{Incompatibility: external},
			})
			return
		}

		r.visit(derived)
		r.lines = append(r.lines, Line{
			Kind:       LineExternalOnly,
			Conclusion: inc,
			Final:      r.isRoot(inc),
			Cause1:     &Cause{Incompatibility: external},
		})
		return
	}

	first, second := r.orderCauses(c1, c2)
	r.lines = append(r.lines, Line{
		Kind:       LineBothExternal,
		Conclusion: inc,
		Final:      r.isRoot(inc),
		Cause1:     &Cause{Incompatibility: first},
		Cause2:     &Cause{Incompatibility: second},
	})
}
