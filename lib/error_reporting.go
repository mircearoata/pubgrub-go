package lib

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
				return "resolving failed"
			}
			if t.Constraint().Inverse().IsEmpty() {
				// Checking if the constraint is "any"
				return fmt.Sprintf("%s is forbidden", t.Dependency())
			}
			return fmt.Sprintf("%s is forbidden", t.String())
		}
		panic("negative term in cause")
	}
	var positive, negative term
	if terms[0].Positive() {
		positive = terms[0]
		negative = terms[1]
	} else {
		positive = terms[1]
		negative = terms[0]
	}
	if positive.Dependency() == state.rootPkg {
		return fmt.Sprintf("installing %s", negative.String())
	}
	return fmt.Sprintf("%s depends on %s", positive.String(), negative.String())
}

func writeErrorMessageRecursive(c *Incompatibility, state *errorGenerationState) {
	if !isDerived(c) {
		return
	}
	c1 := c.Causes()[0]
	c2 := c.Causes()[1]

	if isDerived(c1) && isDerived(c2) {
		l1, ok1 := state.lines[c1]
		l2, ok2 := state.lines[c2]

		if ok1 && ok2 {
			state.writeLine(fmt.Sprintf("Because %s (%d) and %s (%d), %s.", state.causeString(c1), l1, state.causeString(c2), l2, state.causeString(c)))
			return
		}

		if ok1 && !ok2 {
			writeErrorMessageRecursive(c2, state)
			state.writeLine(fmt.Sprintf("And because %s (%d), %s.", state.causeString(c1), l1, state.causeString(c)))
			return
		}

		if !ok1 && ok2 {
			writeErrorMessageRecursive(c1, state)
			state.writeLine(fmt.Sprintf("And because %s (%d), %s.", state.causeString(c2), l2, state.causeString(c)))
		}

		if !isDerived(c1.Causes()[0]) && !isDerived(c1.Causes()[1]) {
			writeErrorMessageRecursive(c2, state)
			writeErrorMessageRecursive(c1, state)
			state.writeLine(fmt.Sprintf("Thus, %s.", state.causeString(c)))
			return
		}

		if !isDerived(c2.Causes()[0]) && !isDerived(c2.Causes()[1]) {
			writeErrorMessageRecursive(c1, state)
			writeErrorMessageRecursive(c2, state)
			state.writeLine(fmt.Sprintf("Thus, %s.", state.causeString(c)))
			return
		}

		writeErrorMessageRecursive(c1, state)
		l := state.tagLastLine(c1)
		state.writeLine("")
		writeErrorMessageRecursive(c2, state)
		state.tagLastLine(c2)
		state.writeLine(fmt.Sprintf("And because %s (%d), %s.", state.causeString(c1), l, state.causeString(c)))
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

			writeErrorMessageRecursive(priorDerived, state)
			state.writeLine(fmt.Sprintf("And because %s, %s.", state.causeString(priorExternal), state.causeString(c)))
			return
		}

		writeErrorMessageRecursive(derived, state)
		state.writeLine(fmt.Sprintf("And because %s, %s.", state.causeString(external), state.causeString(c)))
		return
	}

	// TODO: order the causes as follows "Because x depends on y and y depends on z, x depends on z"
	state.writeLine(fmt.Sprintf("Because %s and %s, %s.", state.causeString(c1), state.causeString(c2), state.causeString(c)))
}

func GetErrorMessage(c *Incompatibility) string {
	state := &errorGenerationState{lines: make(map[*Incompatibility]int), nextLine: 1, result: []string{}, rootPkg: c.Terms()[0].Dependency()}
	writeErrorMessageRecursive(c, state)
	return strings.TrimSpace(strings.Join(state.result, "\n"))
}
