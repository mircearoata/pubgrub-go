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
}

func NewStandardErrorWriter(rootPkg string) *StandardErrorWriter {
	return &StandardErrorWriter{
		nextLine:                   1,
		result:                     []string{},
		lineNumbers:                map[int]int{},
		incompatibilityLineNumbers: map[*Incompatibility]int{},
		rootPkg:                    rootPkg,
	}
}

func (w *StandardErrorWriter) String() string {
	var result []string
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

func (w *StandardErrorWriter) causeString(c *Incompatibility) string {
	if w.isRoot(c) {
		return "version solving failed"
	}
	terms := c.Terms()
	if len(terms) == 1 {
		t := terms[0]
		if t.Positive() {
			if t.Constraint().IsAny() {
				return fmt.Sprintf("%s is forbidden", t.Dependency())
			}
			return fmt.Sprintf("%s is forbidden", t.String())
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
		return fmt.Sprintf("installing %s", dep.String())
	}
	if dep.Constraint().IsEmpty() {
		return fmt.Sprintf("%s forbids %s", pkg.String(), dep.Dependency())
	}
	if dep.Constraint().IsAny() {
		return fmt.Sprintf("%s depends on %s", pkg.String(), dep.Dependency())
	}
	return fmt.Sprintf("%s depends on %s", pkg.String(), dep.String())
}

func (w *StandardErrorWriter) writeLine(line string) {
	w.result = append(w.result, line)
}

func (w *StandardErrorWriter) isRoot(incompatibility *Incompatibility) bool {
	terms := incompatibility.Terms()
	return len(terms) == 1 && terms[0].Positive() && terms[0].Dependency() == w.rootPkg
}

func (w *StandardErrorWriter) WriteLineTwoCauses(cause1, cause2, incompatibility *Incompatibility) {
	if w.isRoot(incompatibility) {
		w.writeLine(fmt.Sprintf("So, because %s and %s, %s.", w.causeString(cause1), w.causeString(cause2), w.causeString(incompatibility)))
	} else {
		w.writeLine(fmt.Sprintf("Because %s and %s, %s.", w.causeString(cause1), w.causeString(cause2), w.causeString(incompatibility)))
	}
}

func (w *StandardErrorWriter) WriteLineTwoCausesOneTag(cause1, cause2, incompatibility *Incompatibility, line2 int) {
	if w.isRoot(incompatibility) {
		w.writeLine(fmt.Sprintf("So, because %s and %s (%d), %s.", w.causeString(cause1), w.causeString(cause2), line2, w.causeString(incompatibility)))
	} else {
		w.writeLine(fmt.Sprintf("Because %s and %s (%d), %s.", w.causeString(cause1), w.causeString(cause2), line2, w.causeString(incompatibility)))
	}
}

func (w *StandardErrorWriter) WriteLineTwoCausesTwoTags(cause1, cause2, incompatibility *Incompatibility, line1, line2 int) {
	if w.isRoot(incompatibility) {
		w.writeLine(fmt.Sprintf("So, because %s (%d) and %s (%d), %s.", w.causeString(cause1), line1, w.causeString(cause2), line2, w.causeString(incompatibility)))
	} else {
		w.writeLine(fmt.Sprintf("Because %s (%d) and %s (%d), %s.", w.causeString(cause1), line1, w.causeString(cause2), line2, w.causeString(incompatibility)))
	}
}

func (w *StandardErrorWriter) WriteLineOneCause(cause, incompatibility *Incompatibility) {
	if w.isRoot(incompatibility) {
		w.writeLine(fmt.Sprintf("So, because %s, %s.", w.causeString(cause), w.causeString(incompatibility)))
	} else {
		w.writeLine(fmt.Sprintf("And because %s, %s.", w.causeString(cause), w.causeString(incompatibility)))
	}
}

func (w *StandardErrorWriter) WriteLineOneCauseOneTag(cause, incompatibility *Incompatibility, line int) {
	if w.isRoot(incompatibility) {
		w.writeLine(fmt.Sprintf("So, because %s (%d), %s.", w.causeString(cause), line, w.causeString(incompatibility)))
	} else {
		w.writeLine(fmt.Sprintf("And because %s (%d), %s.", w.causeString(cause), line, w.causeString(incompatibility)))
	}
}

func (w *StandardErrorWriter) WriteLineNoCause(incompatibility *Incompatibility) {
	w.writeLine(fmt.Sprintf("Thus, %s.", w.causeString(incompatibility)))
}

func (w *StandardErrorWriter) Separate() {
	w.writeLine("")
}
