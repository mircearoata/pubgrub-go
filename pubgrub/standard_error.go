package pubgrub

import (
	"fmt"
	"strings"
)

type ReportLine struct {
	Text    string
	LineNum int
}

type LinesStringer interface {
	LinesString(lines []ReportLine) string
}

type StandardTextReporter struct {
	strings                 StandardCauseStrings
	incompatibilityStringer IncompatibilityStringer
	termStringer            TermStringer
	linesStringer           LinesStringer
}

func NewStandardTextReporter() *StandardTextReporter {
	return &StandardTextReporter{
		strings:                 DefaultCauseStrings,
		incompatibilityStringer: NewStandardIncompatibilityStringer(),
		termStringer:            NewStandardTermStringer(),
		linesStringer:           NewStandardLinesStringer(),
	}
}

func (r *StandardTextReporter) WithStrings(s StandardCauseStrings) *StandardTextReporter {
	r.strings = s
	return r
}

func (r *StandardTextReporter) WithIncompatibilityStringer(s IncompatibilityStringer) *StandardTextReporter {
	r.incompatibilityStringer = s
	return r
}

func (r *StandardTextReporter) WithTermStringer(s TermStringer) *StandardTextReporter {
	r.termStringer = s
	return r
}

func (r *StandardTextReporter) WithLinesStringer(s LinesStringer) *StandardTextReporter {
	r.linesStringer = s
	return r
}

type StandardLinesStringer struct {
	LineNumberFormat string
}

func NewStandardLinesStringer() StandardLinesStringer {
	return StandardLinesStringer{
		LineNumberFormat: "%d. ",
	}
}

func (s StandardLinesStringer) WithLineNumberFormat(format string) StandardLinesStringer {
	s.LineNumberFormat = format
	return s
}

func (s StandardLinesStringer) LinesString(lines []ReportLine) string {
	indent := 0
	for _, l := range lines {
		if l.LineNum > 0 {
			numLen := len(fmt.Sprintf(s.LineNumberFormat, l.LineNum))
			if numLen > indent {
				indent = numLen
			}
		}
	}
	result := make([]string, 0, len(lines))
	for _, l := range lines {
		if l.Text == "" {
			result = append(result, "")
			continue
		}
		prefix := ""
		if l.LineNum > 0 {
			prefix = fmt.Sprintf(s.LineNumberFormat, l.LineNum)
		}
		padding := ""
		if indent > len(prefix) {
			padding = strings.Repeat(" ", indent-len(prefix))
		}
		result = append(result, prefix+padding+l.Text)
	}
	return strings.Join(result, "\n")
}

func (r *StandardTextReporter) Render(report *Report) string {
	if report == nil {
		return ""
	}

	lines := make([]ReportLine, 0, len(report.Lines))
	for _, line := range report.Lines {
		lines = append(lines, ReportLine{
			Text:    r.lineText(line, report.RootPackage),
			LineNum: line.Ref,
		})
	}
	return r.linesStringer.LinesString(lines)
}

func (r *StandardTextReporter) lineText(line Line, rootPkg string) string {
	s := r.strings

	switch line.Kind {
	case LineSeparator:
		return ""

	case LineBothExternal, LineBothReferenced, LineReferencedAndExternal:
		if line.Final {
			return fmt.Sprintf(s.SoBecause, r.twoCausesString(*line.Cause1, *line.Cause2, rootPkg), r.incompatibilityStringer.IncompatibilityString(line.Conclusion, r.termStringer, rootPkg))
		}
		return fmt.Sprintf(s.Because, r.twoCausesString(*line.Cause1, *line.Cause2, rootPkg), r.incompatibilityStringer.IncompatibilityString(line.Conclusion, r.termStringer, rootPkg))

	case LinePriorAndExternal:
		if line.Final {
			return fmt.Sprintf(s.SoBecause, r.twoCausesString(*line.Cause1, *line.Cause2, rootPkg), r.incompatibilityStringer.IncompatibilityString(line.Conclusion, r.termStringer, rootPkg))
		}
		return fmt.Sprintf(s.AndBecause, r.twoCausesString(*line.Cause1, *line.Cause2, rootPkg), r.incompatibilityStringer.IncompatibilityString(line.Conclusion, r.termStringer, rootPkg))

	case LineExternalOnly, LineReferencedOnly:
		if line.Final {
			return fmt.Sprintf(s.SoBecause, r.oneCauseString(*line.Cause1, rootPkg), r.incompatibilityStringer.IncompatibilityString(line.Conclusion, r.termStringer, rootPkg))
		}
		return fmt.Sprintf(s.AndBecause, r.oneCauseString(*line.Cause1, rootPkg), r.incompatibilityStringer.IncompatibilityString(line.Conclusion, r.termStringer, rootPkg))

	case LineNoCauses:
		return fmt.Sprintf(s.Thus, r.incompatibilityStringer.IncompatibilityString(line.Conclusion, r.termStringer, rootPkg))
	}
	return ""
}

func (r *StandardTextReporter) twoCausesString(c1 Cause, c2 Cause, rootPkg string) string {
	return fmt.Sprintf(r.strings.AndCauses, r.oneCauseString(c1, rootPkg), r.oneCauseString(c2, rootPkg))
}

func (r *StandardTextReporter) oneCauseString(c Cause, rootPkg string) string {
	if c.Ref > 0 {
		return fmt.Sprintf(r.strings.CauseRef, r.incompatibilityStringer.IncompatibilityString(c.Incompatibility, r.termStringer, rootPkg), c.Ref)
	}
	return r.incompatibilityStringer.IncompatibilityString(c.Incompatibility, r.termStringer, rootPkg)
}
