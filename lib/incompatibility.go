package lib

type Incompatibility struct {
	terms  map[string]term
	causes []*Incompatibility
}

func (in Incompatibility) Terms() []term {
	var terms []term
	for _, t := range in.terms {
		terms = append(terms, t)
	}
	return terms
}

func (in Incompatibility) Causes() []*Incompatibility {
	return in.causes
}

func (in Incompatibility) get(pkg string) *term {
	if t, ok := in.terms[pkg]; ok {
		return &t
	}
	return nil
}

type setRelation int

const (
	setRelationSatisfied setRelation = iota
	setRelationContradicted
	setRelationAlmostSatisfied
	setRelationInconclusive
)

func (in Incompatibility) relation(ps *partialSolution) (setRelation, *term) {
	result := setRelationSatisfied
	var unsatisfied term

	// TODO: deterministic order
	for _, t := range in.terms {
		t2 := ps.get(t.pkg)
		if t2 != nil {
			rel := t.Relation(*t2)
			if rel == termRelationSatisfied {
				continue
			}
			if rel == termRelationContradicted {
				result = setRelationContradicted
				unsatisfied = t
				break
			}
		}

		// Either term inconclusive, or not present
		if result == setRelationSatisfied {
			result = setRelationAlmostSatisfied
			unsatisfied = t
		} else {
			result = setRelationInconclusive
		}
	}

	if result == setRelationSatisfied || result == setRelationInconclusive {
		return result, nil
	}
	return result, &unsatisfied
}

func (in Incompatibility) makePriorCause(c *Incompatibility, satisfier string) *Incompatibility {
	newIncompatibility := &Incompatibility{
		terms:  make(map[string]term),
		causes: []*Incompatibility{&in, c},
	}
	for _, t := range in.terms {
		if t.pkg != satisfier {
			newIncompatibility.add(t)
		}
	}
	for _, t := range c.Terms() {
		if t.pkg != satisfier {
			newIncompatibility.add(t)
		}
	}
	return newIncompatibility
}

func (in Incompatibility) add(t term) {
	existingTerm := in.get(t.pkg)
	if existingTerm != nil {
		*existingTerm = existingTerm.Intersect(t)
	} else {
		in.terms[t.pkg] = t
	}
}
