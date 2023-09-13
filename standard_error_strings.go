package pubgrub

type StandardErrorStaticStrings struct {
	TwoCauses             string
	TwoCausesFinal        string
	TwoCausesOneTag       string
	TwoCausesOneTagFinal  string
	TwoCausesTwoTags      string
	TwoCausesTwoTagsFinal string
	OneCause              string
	OneCauseFinal         string
	OneCauseOneTag        string
	OneCauseOneTagFinal   string
	NoCause               string

	ResolvingFailed string

	DependsOn   string
	Installing  string
	Forbids     string
	IsForbidden string
}

var DefaultStandardErrorStrings = StandardErrorStaticStrings{
	TwoCauses:             "Because %s and %s, %s.",
	TwoCausesFinal:        "So, because %s and %s, %s.",
	TwoCausesOneTag:       "Because %s and %s (%d), %s.",
	TwoCausesOneTagFinal:  "So, because %s and %s (%d), %s.",
	TwoCausesTwoTags:      "Because %s (%d) and %s (%d), %s.",
	TwoCausesTwoTagsFinal: "So, because %s (%d) and %s (%d), %s.",
	OneCause:              "And because %s, %s.",
	OneCauseFinal:         "So, because %s, %s.",
	OneCauseOneTag:        "And because %s (%d), %s.",
	OneCauseOneTagFinal:   "So, because %s (%d), %s.",
	NoCause:               "Thus, %s.",

	ResolvingFailed: "version solving failed",

	DependsOn:   "%s depends on %s",
	Installing:  "installing %s",
	Forbids:     "%s forbids %s",
	IsForbidden: "%s is forbidden",
}

type StandardErrorStringer interface {
	Term(t Term, includeVersion bool) string
}

type DefaultStandardErrorStringer struct{}

func (w DefaultStandardErrorStringer) Term(t Term, includeVersion bool) string {
	if includeVersion {
		return t.String()
	}
	return t.Dependency()
}
