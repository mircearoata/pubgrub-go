package pubgrub

type SolvingErrorWriter interface {
	TagLastLine(incompatibility *Incompatibility) int
	GetTag(incompatibility *Incompatibility) (int, bool)
	WriteLineTwoCauses(cause1, cause2, incompatibility *Incompatibility)
	WriteLineTwoCausesOneTag(cause1, cause2, incompatibility *Incompatibility, line2 int)
	WriteLineTwoCausesTwoTags(cause1, cause2, incompatibility *Incompatibility, line1, line2 int)
	WriteLineOneCause(cause, incompatibility *Incompatibility)
	WriteLineOneCauseOneTag(cause, incompatibility *Incompatibility, line int)
	WriteLineNoCause(incompatibility *Incompatibility)
	Separate()
}

type IncompatibilityStringer interface {
	IncompatibilityString(incompatibility *Incompatibility, rootPkg string) string
}

type TermStringer interface {
	Term(t Term, includeVersion bool) string
}

type SolvingError struct {
	cause *Incompatibility
}

func (e SolvingError) Cause() *Incompatibility {
	return e.cause
}

func (e SolvingError) Error() string {
	// A solving error is only caused by a root incompatibility
	rootPkg := e.cause.Terms()[0].Dependency()
	writer := NewStandardErrorWriter(rootPkg)
	e.WriteTo(writer)
	return writer.String()
}

func (e SolvingError) WriteTo(writer SolvingErrorWriter) {
	writeErrorMessageRecursive(e.cause, writer)
}

func isDerived(c *Incompatibility) bool {
	return len(c.Causes()) == 2
}

func writeErrorMessageRecursive(c *Incompatibility, writer SolvingErrorWriter) {
	if !isDerived(c) {
		return
	}
	c1 := c.Causes()[0]
	c2 := c.Causes()[1]

	if isDerived(c1) && isDerived(c2) {
		l1, ok1 := writer.GetTag(c1)
		l2, ok2 := writer.GetTag(c2)

		if ok1 && ok2 {
			var first, second *Incompatibility
			var firstLine, secondLine int
			if l1 < l2 {
				first = c1
				second = c2
				firstLine = l1
				secondLine = l2
			} else {
				first = c2
				second = c1
				firstLine = l2
				secondLine = l1
			}
			writer.WriteLineTwoCausesTwoTags(first, second, c, firstLine, secondLine)
			return
		}

		if ok1 || ok2 {
			var short, long *Incompatibility
			var longLine int
			if ok1 && !ok2 {
				short = c2
				long = c1
				longLine = l1
			} else if !ok1 && ok2 {
				short = c1
				long = c2
				longLine = l2
			}

			writeErrorMessageRecursive(short, writer)
			writer.WriteLineOneCauseOneTag(long, c, longLine)
			return
		}

		if !isDerived(c1.Causes()[0]) && !isDerived(c1.Causes()[1]) {
			writeErrorMessageRecursive(c2, writer)
			writeErrorMessageRecursive(c1, writer)
			writer.WriteLineNoCause(c)
			return
		}

		if !isDerived(c2.Causes()[0]) && !isDerived(c2.Causes()[1]) {
			writeErrorMessageRecursive(c1, writer)
			writeErrorMessageRecursive(c2, writer)
			writer.WriteLineNoCause(c)
			return
		}

		var first, second *Incompatibility
		// This isn't the best way to do this, but it's the easiest
		if c1.Terms()[0].Constraint().String() < c2.Terms()[0].Constraint().String() {
			first = c1
			second = c2
		} else {
			first = c2
			second = c1
		}
		writeErrorMessageRecursive(first, writer)
		l := writer.TagLastLine(first)
		writer.Separate()
		writeErrorMessageRecursive(second, writer)
		writer.TagLastLine(second)
		writer.WriteLineOneCauseOneTag(first, c, l)
		return
	}

	if (!isDerived(c1) && isDerived(c2)) || (isDerived(c1) && !isDerived(c2)) {
		var derived, external *Incompatibility
		if isDerived(c1) {
			derived = c1
			external = c2
		} else {
			derived = c2
			external = c1
		}

		l, ok := writer.GetTag(derived)

		if ok {
			writer.WriteLineTwoCausesOneTag(external, derived, c, l)
			return
		}

		dc1 := derived.Causes()[0]
		dc2 := derived.Causes()[1]

		if (!isDerived(dc1) && isDerived(dc2)) || (isDerived(dc1) && !isDerived(dc2)) {
			var priorDerived, priorExternal *Incompatibility
			if isDerived(dc1) {
				priorDerived = dc1
				priorExternal = dc2
			} else {
				priorDerived = dc2
				priorExternal = dc1
			}

			writeErrorMessageRecursive(priorDerived, writer)
			writer.WriteLineOneCause(priorExternal, c)
			return
		}

		writeErrorMessageRecursive(derived, writer)
		writer.WriteLineOneCause(external, c)
		return
	}

	sharedPkg := ""
	for _, t1 := range c1.Terms() {
		for _, t2 := range c2.Terms() {
			if t1.Dependency() == t2.Dependency() {
				sharedPkg = t1.Dependency()
				break
			}
		}
	}
	var first, second *Incompatibility
	// The negative term is the package that is depended on
	// Therefore we want the first incompatibility to be the one that has a negative term of the shared package
	if !c1.get(sharedPkg).Positive() {
		first = c1
		second = c2
	} else {
		first = c2
		second = c1
	}

	writer.WriteLineTwoCauses(first, second, c)
}
