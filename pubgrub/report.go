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

type ReportCause struct {
	Incompatibility *Incompatibility
	Ref             int
}

type Line struct {
	Kind       LineKind
	Conclusion *Incompatibility
	Ref        int
	Final      bool
	Cause1     *ReportCause
	Cause2     *ReportCause
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
		root:        cause,
	}
	b.countDerivations(cause)
	b.visit(cause, false)
	report.Lines = b.lines
	return report
}

type reportBuilder struct {
	rootPkg     string
	derivations map[*Incompatibility]int
	lineNumbers map[*Incompatibility]int
	lines       []Line
	nextLine    int
	root        *Incompatibility
}

func (r *reportBuilder) countDerivations(inc *Incompatibility) {
	if inc == nil {
		return
	}
	r.derivations[inc]++
	if r.derivations[inc] == 1 {
		cause := inc.Cause()
		if conflictCause, ok := cause.(ConflictCause); ok {
			r.countDerivations(conflictCause.A)
			r.countDerivations(conflictCause.B)
		}
	}
}

func (r *reportBuilder) isSingleLine(cause ConflictCause) bool {
	_, aIsConflict := cause.A.Cause().(ConflictCause)
	_, bIsConflict := cause.B.Cause().(ConflictCause)
	return !aIsConflict && !bIsConflict
}

func (r *reportBuilder) number(inc *Incompatibility, numbered bool) int {
	if !numbered {
		return 0
	}
	number := r.nextLine
	r.lineNumbers[inc] = number
	r.nextLine++
	return number
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

func (r *reportBuilder) visit(inc *Incompatibility, conclusion bool) {
	cause := inc.Cause()
	conflictCause, ok := cause.(ConflictCause)
	if !ok {
		return
	}
	c1, c2 := conflictCause.A, conflictCause.B

	numbered := conclusion || r.derivations[inc] > 1
	isFinal := conclusion || inc == r.root

	c1ConflictCause, c1IsConflict := c1.Cause().(ConflictCause)
	c2ConflictCause, c2IsConflict := c2.Cause().(ConflictCause)

	if c1IsConflict && c2IsConflict {
		l1, ok1 := r.lineNumbers[c1]
		l2, ok2 := r.lineNumbers[c2]

		if ok1 && ok2 {
			first, second := r.orderCauses(c1, c2)
			line1 := r.lineNumbers[first]
			line2 := r.lineNumbers[second]
			r.lines = append(r.lines, Line{
				Kind:       LineBothReferenced,
				Conclusion: inc,
				Ref:        r.number(inc, numbered),
				Final:      isFinal,
				Cause1:     &ReportCause{Incompatibility: first, Ref: line1},
				Cause2:     &ReportCause{Incompatibility: second, Ref: line2},
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
			r.visit(withoutLine, false)
			r.lines = append(r.lines, Line{
				Kind:       LineReferencedOnly,
				Conclusion: inc,
				Ref:        r.number(inc, numbered),
				Final:      isFinal,
				Cause1:     &ReportCause{Incompatibility: withLine, Ref: line},
			})
			return
		}

		single1 := r.isSingleLine(c1ConflictCause)
		single2 := r.isSingleLine(c2ConflictCause)
		if single1 || single2 {
			var first, second *Incompatibility
			if single2 {
				first, second = c1, c2
			} else {
				first, second = c2, c1
			}
			r.visit(first, false)
			r.visit(second, false)
			r.lines = append(r.lines, Line{
				Kind:       LineNoCauses,
				Conclusion: inc,
				Ref:        r.number(inc, numbered),
				Final:      isFinal,
			})
			return
		}

		first, second := r.orderCauses(c1, c2)
		r.visit(first, true)
		r.lines = append(r.lines, Line{
			Kind: LineSeparator,
		})
		r.visit(second, false)
		firstLine := r.lineNumbers[first]
		r.lines = append(r.lines, Line{
			Kind:       LineReferencedOnly,
			Conclusion: inc,
			Ref:        r.number(inc, numbered),
			Final:      isFinal,
			Cause1:     &ReportCause{Incompatibility: first, Ref: firstLine},
		})
		return
	}

	if c1IsConflict != c2IsConflict {
		derived, external := c1, c2
		derivedConflictCause := c1ConflictCause
		if !c1IsConflict {
			derived, external = c2, c1
			derivedConflictCause = c2ConflictCause
		}

		if derivedLine, ok := r.lineNumbers[derived]; ok {
			r.lines = append(r.lines, Line{
				Kind:       LineReferencedAndExternal,
				Conclusion: inc,
				Ref:        r.number(inc, numbered),
				Final:      isFinal,
				Cause1:     &ReportCause{Incompatibility: external},
				Cause2:     &ReportCause{Incompatibility: derived, Ref: derivedLine},
			})
			return
		}

		dc1, dc2 := derivedConflictCause.A, derivedConflictCause.B
		_, dc1IsConflict := dc1.Cause().(ConflictCause)
		_, dc2IsConflict := dc2.Cause().(ConflictCause)
		_, dc1HasLine := r.lineNumbers[dc1]
		_, dc2HasLine := r.lineNumbers[dc2]

		if r.derivations[derived] <= 1 && ((dc1IsConflict && !dc2IsConflict && !dc1HasLine) || (dc2IsConflict && !dc1IsConflict && !dc2HasLine)) {
			var priorDerived, priorExternal *Incompatibility
			if dc1IsConflict {
				priorDerived, priorExternal = dc1, dc2
			} else {
				priorDerived, priorExternal = dc2, dc1
			}
			r.visit(priorDerived, false)
			r.lines = append(r.lines, Line{
				Kind:       LinePriorAndExternal,
				Conclusion: inc,
				Ref:        r.number(inc, numbered),
				Final:      isFinal,
				Cause1:     &ReportCause{Incompatibility: priorExternal},
				Cause2:     &ReportCause{Incompatibility: external},
			})
			return
		}

		r.visit(derived, false)
		r.lines = append(r.lines, Line{
			Kind:       LineExternalOnly,
			Conclusion: inc,
			Ref:        r.number(inc, numbered),
			Final:      isFinal,
			Cause1:     &ReportCause{Incompatibility: external},
		})
		return
	}

	first, second := r.orderCauses(c1, c2)
	r.lines = append(r.lines, Line{
		Kind:       LineBothExternal,
		Conclusion: inc,
		Ref:        r.number(inc, numbered),
		Final:      isFinal,
		Cause1:     &ReportCause{Incompatibility: first},
		Cause2:     &ReportCause{Incompatibility: second},
	})
}
