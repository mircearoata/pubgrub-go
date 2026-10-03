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

func (r *StandardTextReporter) twoCausesString(c1 ReportCause, c2 ReportCause, rootPkg string) string {
	if res, ok := r.requiresBoth(c1, c2, rootPkg); ok {
		return res
	}
	if res, ok := r.requiresThrough(c1, c2, rootPkg); ok {
		return res
	}
	if res, ok := r.requiresForbidden(c1, c2, rootPkg); ok {
		return res
	}
	return fmt.Sprintf(r.strings.AndCauses, r.oneCauseString(c1, rootPkg), r.oneCauseString(c2, rootPkg))
}

func (r *StandardTextReporter) oneCauseString(c ReportCause, rootPkg string) string {
	if c.Ref > 0 {
		return fmt.Sprintf(r.strings.CauseRef, r.incompatibilityStringer.IncompatibilityString(c.Incompatibility, r.termStringer, rootPkg), c.Ref)
	}
	return r.incompatibilityStringer.IncompatibilityString(c.Incompatibility, r.termStringer, rootPkg)
}

func LinksTo(prior, latter *Incompatibility) bool {
	if prior == nil || latter == nil {
		return false
	}
	neg, okNeg := prior.SingleNegative()
	if !okNeg {
		return false
	}
	pos, okPos := latter.SinglePositive()
	if !okPos {
		return false
	}
	return neg.Dependency() == pos.Dependency() &&
		neg.Constraint().Difference(pos.Constraint()).IsEmpty()
}

type RequiresBothMatch struct {
	Subject   Term
	Negative1 []Term
	Negative2 []Term
	Ref1      int
	Ref2      int
}

func MatchRequiresBoth(c1, c2 ReportCause) (*RequiresBothMatch, bool) {
	if c1.Incompatibility.Len() <= 1 || c2.Incompatibility.Len() <= 1 {
		return nil, false
	}
	pos1, ok1 := c1.Incompatibility.SinglePositive()
	pos2, ok2 := c2.Incompatibility.SinglePositive()
	if !ok1 || !ok2 {
		return nil, false
	}
	if !pos1.Equal(pos2) {
		return nil, false
	}

	neg1 := c1.Incompatibility.Negatives()
	neg2 := c2.Incompatibility.Negatives()

	return &RequiresBothMatch{
		Subject:   pos1,
		Negative1: neg1,
		Negative2: neg2,
		Ref1:      c1.Ref,
		Ref2:      c2.Ref,
	}, true
}

type RequiresThroughMatch struct {
	Prior  ReportCause
	Latter ReportCause
}

func MatchRequiresThrough(c1, c2 ReportCause) (*RequiresThroughMatch, bool) {
	if c1.Incompatibility.Len() <= 1 || c2.Incompatibility.Len() <= 1 {
		return nil, false
	}

	if LinksTo(c1.Incompatibility, c2.Incompatibility) {
		return &RequiresThroughMatch{Prior: c1, Latter: c2}, true
	}
	if LinksTo(c2.Incompatibility, c1.Incompatibility) {
		return &RequiresThroughMatch{Prior: c2, Latter: c1}, true
	}
	return nil, false
}

type RequiresForbiddenMatch struct {
	Prior     ReportCause
	Forbidden ReportCause
}

func MatchRequiresForbidden(c1, c2 ReportCause) (*RequiresForbiddenMatch, bool) {
	if c1.Incompatibility.Len() != 1 && c2.Incompatibility.Len() != 1 {
		return nil, false
	}

	prior, latter := c1, c2
	if c1.Incompatibility.Len() == 1 {
		prior, latter = c2, c1
	}

	if !LinksTo(prior.Incompatibility, latter.Incompatibility) {
		return nil, false
	}

	return &RequiresForbiddenMatch{
		Prior:     prior,
		Forbidden: latter,
	}, true
}

func (r *StandardTextReporter) formatNegatives(terms []Term, ref int) string {
	str := strings.Join(FormatTerms(terms, r.termStringer, false), r.strings.Alternative)
	if ref > 0 {
		str = fmt.Sprintf(r.strings.CauseRef, str, ref)
	}
	return str
}

func (r *StandardTextReporter) requiresBoth(c1 ReportCause, c2 ReportCause, rootPkg string) (string, bool) {
	match, ok := MatchRequiresBoth(c1, c2)
	if !ok {
		return "", false
	}

	neg1Str := r.formatNegatives(match.Negative1, match.Ref1)
	neg2Str := r.formatNegatives(match.Negative2, match.Ref2)

	if match.Subject.Dependency() == rootPkg {
		return fmt.Sprintf(r.strings.InstallingBoth, neg1Str, neg2Str), true
	}
	_, isDependency1 := c1.Incompatibility.Cause().(DependencyCause)
	_, isDependency2 := c2.Incompatibility.Cause().(DependencyCause)
	if isDependency1 && isDependency2 {
		return fmt.Sprintf(r.strings.DependsOnBoth, FormatTerm(match.Subject, r.termStringer, true), neg1Str, neg2Str), true
	}
	return fmt.Sprintf(r.strings.RequiresBoth, FormatTerm(match.Subject, r.termStringer, true), neg1Str, neg2Str), true
}

func (r *StandardTextReporter) requiresThrough(c1 ReportCause, c2 ReportCause, rootPkg string) (string, bool) {
	match, ok := MatchRequiresThrough(c1, c2)
	if !ok {
		return "", false
	}

	negString := r.formatNegatives(match.Latter.Incompatibility.Negatives(), match.Latter.Ref)

	_, isDependency := match.Latter.Incompatibility.Cause().(DependencyCause)
	if isDependency {
		return fmt.Sprintf(r.strings.WhichDependsOn, r.oneCauseString(match.Prior, rootPkg), negString), true
	}
	return fmt.Sprintf(r.strings.WhichRequires, r.oneCauseString(match.Prior, rootPkg), negString), true
}

func (r *StandardTextReporter) requiresForbidden(c1 ReportCause, c2 ReportCause, rootPkg string) (string, bool) {
	match, ok := MatchRequiresForbidden(c1, c2)
	if !ok {
		return "", false
	}

	priorStr := r.oneCauseString(match.Prior, rootPkg)
	return r.whichCauseString(priorStr, match.Forbidden), true
}

func (r *StandardTextReporter) whichCauseString(priorStr string, latter ReportCause) string {
	if _, ok := latter.Incompatibility.Cause().(NoVersionsCause); ok {
		return fmt.Sprintf(r.strings.WhichNoVersions, priorStr)
	}
	res := fmt.Sprintf(r.strings.WhichIsForbidden, priorStr)
	if latter.Ref > 0 {
		res = fmt.Sprintf(r.strings.CauseRef, res, latter.Ref)
	}
	return res
}
