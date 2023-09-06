package pubgrub

import (
	"fmt"
	"strings"
)

type errorGenerationState struct {
	rootPkg  string
	lines    map[*Incompatibility]int
	nextLine int
	result   []string
}

func (state *errorGenerationState) tagLastLine(c *Incompatibility) int {
	l := state.nextLine
	state.lines[c] = state.nextLine
	state.nextLine++
	state.result[len(state.result)-1] = fmt.Sprintf("%d. %s", l, state.result[len(state.result)-1])
	return l
}

func (state *errorGenerationState) writeLine(s string) {
	state.result = append(state.result, s)
}

func isDerived(c *Incompatibility) bool {
	return len(c.Causes()) == 2
}

func (state *errorGenerationState) causeString(c *Incompatibility) string {
	terms := c.Terms()
	if len(terms) == 1 {
		t := terms[0]
		if t.Positive() {
			if t.Dependency() == state.rootPkg {
				return "version solving failed"
			}
			if t.Constraint().IsAny() {
				return fmt.Sprintf("%s is forbidden", t.Dependency())
			}
			return fmt.Sprintf("%s is forbidden", t.String())
		}
		panic("negative term in cause")
	}
	var pkg, dep term
	if terms[0].Positive() {
		pkg = terms[0]
		dep = terms[1]
	} else {
		pkg = terms[1]
		dep = terms[0]
	}
	if dep.Positive() {
		// This is an optional dependency, which has a positive term, but with an inverse constraint
		// We revert the constraint here to get the term in a similar format to the others
		dep = term{pkg: dep.Dependency(), versionConstraint: dep.Constraint().Inverse(), positive: false}
	}
	if pkg.Dependency() == state.rootPkg {
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

func writeErrorMessageRecursive(c *Incompatibility, state *errorGenerationState, isFirst bool) {
	if !isDerived(c) {
		return
	}
	c1 := c.Causes()[0]
	c2 := c.Causes()[1]

	if isDerived(c1) && isDerived(c2) {
		l1, ok1 := state.lines[c1]
		l2, ok2 := state.lines[c2]

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
			state.writeLine(fmt.Sprintf("Because %s (%d) and %s (%d), %s.", state.causeString(first), firstLine, state.causeString(second), secondLine, state.causeString(c)))
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

			writeErrorMessageRecursive(short, state, false)
			if isFirst {
				state.writeLine(fmt.Sprintf("So, because %s (%d), %s.", state.causeString(long), longLine, state.causeString(c)))
			} else {
				state.writeLine(fmt.Sprintf("And because %s (%d), %s.", state.causeString(long), longLine, state.causeString(c)))
			}
			return
		}

		if !isDerived(c1.Causes()[0]) && !isDerived(c1.Causes()[1]) {
			writeErrorMessageRecursive(c2, state, false)
			writeErrorMessageRecursive(c1, state, false)
			state.writeLine(fmt.Sprintf("Thus, %s.", state.causeString(c)))
			return
		}

		if !isDerived(c2.Causes()[0]) && !isDerived(c2.Causes()[1]) {
			writeErrorMessageRecursive(c1, state, false)
			writeErrorMessageRecursive(c2, state, false)
			state.writeLine(fmt.Sprintf("Thus, %s.", state.causeString(c)))
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
		writeErrorMessageRecursive(first, state, false)
		l := state.tagLastLine(first)
		state.writeLine("")
		writeErrorMessageRecursive(second, state, false)
		state.tagLastLine(second)
		if isFirst {
			state.writeLine(fmt.Sprintf("So, because %s (%d), %s.", state.causeString(first), l, state.causeString(c)))
		} else {
			state.writeLine(fmt.Sprintf("And because %s (%d), %s.", state.causeString(first), l, state.causeString(c)))
		}
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

		l, ok := state.lines[derived]

		if ok {
			state.writeLine(fmt.Sprintf("Because %s and %s (%d), %s.", state.causeString(external), state.causeString(derived), l, state.causeString(c)))
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

			writeErrorMessageRecursive(priorDerived, state, false)
			if isFirst {
				state.writeLine(fmt.Sprintf("So, because %s, %s.", state.causeString(priorExternal), state.causeString(c)))
			} else {
				state.writeLine(fmt.Sprintf("And because %s, %s.", state.causeString(priorExternal), state.causeString(c)))
			}
			return
		}

		writeErrorMessageRecursive(derived, state, false)
		if isFirst {
			state.writeLine(fmt.Sprintf("So, because %s, %s.", state.causeString(external), state.causeString(c)))
		} else {
			state.writeLine(fmt.Sprintf("And because %s, %s.", state.causeString(external), state.causeString(c)))
		}
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

	state.writeLine(fmt.Sprintf("Because %s and %s, %s.", state.causeString(first), state.causeString(second), state.causeString(c)))
}

func GetErrorMessage(c *Incompatibility) string {
	state := &errorGenerationState{lines: make(map[*Incompatibility]int), nextLine: 1, result: []string{}, rootPkg: c.Terms()[0].Dependency()}
	writeErrorMessageRecursive(c, state, true)
	return strings.TrimSpace(strings.Join(state.result, "\n"))
}
