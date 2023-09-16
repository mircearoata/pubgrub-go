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
	strings                    StandardCauseStrings
	incompatibilityStringer    IncompatibilityStringer
}

func NewStandardErrorWriter(rootPkg string) *StandardErrorWriter {
	w := &StandardErrorWriter{
		nextLine:                   1,
		result:                     []string{},
		lineNumbers:                map[int]int{},
		incompatibilityLineNumbers: map[*Incompatibility]int{},
		rootPkg:                    rootPkg,
		strings:                    DefaultCauseStrings,
		incompatibilityStringer:    NewStandardIncompatibilityStringer(),
	}
	return w
}

func (w *StandardErrorWriter) WithStrings(s StandardCauseStrings) *StandardErrorWriter {
	w.strings = s
	return w
}

func (w *StandardErrorWriter) WithIncompatibilityStringer(s IncompatibilityStringer) *StandardErrorWriter {
	w.incompatibilityStringer = s
	return w
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

func (w *StandardErrorWriter) WriteLine(line string) {
	w.result = append(w.result, line)
}

func (w *StandardErrorWriter) IsRoot(incompatibility *Incompatibility) bool {
	terms := incompatibility.Terms()
	return len(terms) == 1 && terms[0].Positive() && terms[0].Dependency() == w.rootPkg
}

func (w *StandardErrorWriter) WriteLineTwoCauses(cause1, cause2, incompatibility *Incompatibility) {
	if w.IsRoot(incompatibility) {
		w.WriteLine(fmt.Sprintf(w.strings.TwoCausesFinal, w.incompatibilityStringer.IncompatibilityString(cause1, w.rootPkg), w.incompatibilityStringer.IncompatibilityString(cause2, w.rootPkg), w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
	} else {
		w.WriteLine(fmt.Sprintf(w.strings.TwoCauses, w.incompatibilityStringer.IncompatibilityString(cause1, w.rootPkg), w.incompatibilityStringer.IncompatibilityString(cause2, w.rootPkg), w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
	}
}

func (w *StandardErrorWriter) WriteLineTwoCausesOneTag(cause1, cause2, incompatibility *Incompatibility, line2 int) {
	if w.IsRoot(incompatibility) {
		w.WriteLine(fmt.Sprintf(w.strings.TwoCausesOneTagFinal, w.incompatibilityStringer.IncompatibilityString(cause1, w.rootPkg), w.incompatibilityStringer.IncompatibilityString(cause2, w.rootPkg), line2, w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
	} else {
		w.WriteLine(fmt.Sprintf(w.strings.TwoCausesOneTag, w.incompatibilityStringer.IncompatibilityString(cause1, w.rootPkg), w.incompatibilityStringer.IncompatibilityString(cause2, w.rootPkg), line2, w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
	}
}

func (w *StandardErrorWriter) WriteLineTwoCausesTwoTags(cause1, cause2, incompatibility *Incompatibility, line1, line2 int) {
	if w.IsRoot(incompatibility) {
		w.WriteLine(fmt.Sprintf(w.strings.TwoCausesTwoTagsFinal, w.incompatibilityStringer.IncompatibilityString(cause1, w.rootPkg), line1, w.incompatibilityStringer.IncompatibilityString(cause2, w.rootPkg), line2, w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
	} else {
		w.WriteLine(fmt.Sprintf(w.strings.TwoCausesTwoTags, w.incompatibilityStringer.IncompatibilityString(cause1, w.rootPkg), line1, w.incompatibilityStringer.IncompatibilityString(cause2, w.rootPkg), line2, w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
	}
}

func (w *StandardErrorWriter) WriteLineOneCause(cause, incompatibility *Incompatibility) {
	if w.IsRoot(incompatibility) {
		w.WriteLine(fmt.Sprintf(w.strings.OneCauseFinal, w.incompatibilityStringer.IncompatibilityString(cause, w.rootPkg), w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
	} else {
		w.WriteLine(fmt.Sprintf(w.strings.OneCause, w.incompatibilityStringer.IncompatibilityString(cause, w.rootPkg), w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
	}
}

func (w *StandardErrorWriter) WriteLineOneCauseOneTag(cause, incompatibility *Incompatibility, line int) {
	if w.IsRoot(incompatibility) {
		w.WriteLine(fmt.Sprintf(w.strings.OneCauseOneTagFinal, w.incompatibilityStringer.IncompatibilityString(cause, w.rootPkg), line, w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
	} else {
		w.WriteLine(fmt.Sprintf(w.strings.OneCauseOneTag, w.incompatibilityStringer.IncompatibilityString(cause, w.rootPkg), line, w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
	}
}

func (w *StandardErrorWriter) WriteLineNoCause(incompatibility *Incompatibility) {
	w.WriteLine(fmt.Sprintf(w.strings.NoCause, w.incompatibilityStringer.IncompatibilityString(incompatibility, w.rootPkg)))
}

func (w *StandardErrorWriter) Separate() {
	w.WriteLine("")
}
