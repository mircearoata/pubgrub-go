package pubgrub

import (
	"fmt"
	"strings"
)

type StandardErrorWriter struct {
	nextLine                   int
	result                     []string
	lineNumbers                map[int]int
	incompatibilityLineNumbers map[*Incompatibility]int
	rootPkg                    string
	staticStrings              StandardErrorStaticStrings
	stringer                   StandardErrorStringer
}

func NewStandardErrorWriter(rootPkg string) *StandardErrorWriter {
	return &StandardErrorWriter{
		nextLine:                   1,
		result:                     []string{},
		lineNumbers:                map[int]int{},
		incompatibilityLineNumbers: map[*Incompatibility]int{},
		rootPkg:                    rootPkg,
		staticStrings:              DefaultStandardErrorStrings,
		stringer:                   DefaultStandardErrorStringer{},
	}
}

func (w *StandardErrorWriter) SetStaticStrings(s StandardErrorStaticStrings) {
	w.staticStrings = s
}

func (w *StandardErrorWriter) SetStringer(s StandardErrorStringer) {
	w.stringer = s
}

func (w *StandardErrorWriter) String() string {
	result := make([]string, 0, len(w.result))
	for i, line := range w.result {
		if line == "" {
			result = append(result, "")
			continue
		}
		indent := 0
		for _, num := range w.lineNumbers {
			lineNumLen := len(fmt.Sprintf("%d. ", num))
			if lineNumLen > indent {
				indent = lineNumLen
			}
		}
		lineNum := ""
		if num, ok := w.lineNumbers[i]; ok {
			lineNum = fmt.Sprintf("%d. ", num)
		}
		result = append(result, fmt.Sprintf("%s%s%s", lineNum, strings.Repeat(" ", indent-len(lineNum)), line))
	}
	return strings.Join(result, "\n")
}

func (w *StandardErrorWriter) TagLastLine(incompatibility *Incompatibility) int {
	l := w.nextLine
	w.nextLine++
	w.lineNumbers[len(w.result)-1] = l
	w.incompatibilityLineNumbers[incompatibility] = l
	return l
}

func (w *StandardErrorWriter) GetTag(incompatibility *Incompatibility) (int, bool) {
	if l, ok := w.incompatibilityLineNumbers[incompatibility]; ok {
		return l, true
	}
	return 0, false
}

func (w *StandardErrorWriter) CauseString(c *Incompatibility) string {
	if w.IsRoot(c) {
		return w.staticStrings.ResolvingFailed
	}
	terms := c.Terms()
	if len(terms) == 1 {
		t := terms[0]
		if t.Positive() {
			if t.Constraint().IsAny() {
				return fmt.Sprintf(w.staticStrings.IsForbidden, w.stringer.Term(t, false))
			}
			return fmt.Sprintf(w.staticStrings.IsForbidden, w.stringer.Term(t, true))
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
	if pkg.Dependency() == w.rootPkg {
		return fmt.Sprintf(w.staticStrings.Installing, w.stringer.Term(dep, true))
	}
	if dep.Constraint().IsEmpty() {
		return fmt.Sprintf(w.staticStrings.Forbids, w.stringer.Term(pkg, true), w.stringer.Term(dep, false))
	}
	if dep.Constraint().IsAny() {
		return fmt.Sprintf(w.staticStrings.DependsOn, w.stringer.Term(pkg, true), w.stringer.Term(dep, false))
	}
	return fmt.Sprintf(w.staticStrings.DependsOn, w.stringer.Term(pkg, true), w.stringer.Term(dep, true))
}

func (w *StandardErrorWriter) WriteLine(line string) {
	w.result = append(w.result, line)
}

func (w *StandardErrorWriter) IsRoot(incompatibility *Incompatibility) bool {
	terms := incompatibility.Terms()
	return len(terms) == 1 && terms[0].Positive() && terms[0].Dependency() == w.rootPkg
}

func (w *StandardErrorWriter) WriteLineTwoCauses(cause1, cause2, incompatibility *Incompatibility) {
	if w.IsRoot(incompatibility) {
		w.WriteLine(fmt.Sprintf(w.staticStrings.TwoCausesFinal, w.CauseString(cause1), w.CauseString(cause2), w.CauseString(incompatibility)))
	} else {
		w.WriteLine(fmt.Sprintf(w.staticStrings.TwoCauses, w.CauseString(cause1), w.CauseString(cause2), w.CauseString(incompatibility)))
	}
}

func (w *StandardErrorWriter) WriteLineTwoCausesOneTag(cause1, cause2, incompatibility *Incompatibility, line2 int) {
	if w.IsRoot(incompatibility) {
		w.WriteLine(fmt.Sprintf(w.staticStrings.TwoCausesOneTagFinal, w.CauseString(cause1), w.CauseString(cause2), line2, w.CauseString(incompatibility)))
	} else {
		w.WriteLine(fmt.Sprintf(w.staticStrings.TwoCausesOneTag, w.CauseString(cause1), w.CauseString(cause2), line2, w.CauseString(incompatibility)))
	}
}

func (w *StandardErrorWriter) WriteLineTwoCausesTwoTags(cause1, cause2, incompatibility *Incompatibility, line1, line2 int) {
	if w.IsRoot(incompatibility) {
		w.WriteLine(fmt.Sprintf(w.staticStrings.TwoCausesTwoTagsFinal, w.CauseString(cause1), line1, w.CauseString(cause2), line2, w.CauseString(incompatibility)))
	} else {
		w.WriteLine(fmt.Sprintf(w.staticStrings.TwoCausesTwoTags, w.CauseString(cause1), line1, w.CauseString(cause2), line2, w.CauseString(incompatibility)))
	}
}

func (w *StandardErrorWriter) WriteLineOneCause(cause, incompatibility *Incompatibility) {
	if w.IsRoot(incompatibility) {
		w.WriteLine(fmt.Sprintf(w.staticStrings.OneCauseFinal, w.CauseString(cause), w.CauseString(incompatibility)))
	} else {
		w.WriteLine(fmt.Sprintf(w.staticStrings.OneCause, w.CauseString(cause), w.CauseString(incompatibility)))
	}
}

func (w *StandardErrorWriter) WriteLineOneCauseOneTag(cause, incompatibility *Incompatibility, line int) {
	if w.IsRoot(incompatibility) {
		w.WriteLine(fmt.Sprintf(w.staticStrings.OneCauseOneTagFinal, w.CauseString(cause), line, w.CauseString(incompatibility)))
	} else {
		w.WriteLine(fmt.Sprintf(w.staticStrings.OneCauseOneTag, w.CauseString(cause), line, w.CauseString(incompatibility)))
	}
}

func (w *StandardErrorWriter) WriteLineNoCause(incompatibility *Incompatibility) {
	w.WriteLine(fmt.Sprintf(w.staticStrings.NoCause, w.CauseString(incompatibility)))
}

func (w *StandardErrorWriter) Separate() {
	w.WriteLine("")
}
