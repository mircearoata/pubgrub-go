package pubgrub

type SolvingError struct {
	cause   *Incompatibility
	rootPkg string
}

func (e SolvingError) Cause() *Incompatibility {
	return e.cause
}

func (e SolvingError) RootPackage() string {
	return e.rootPkg
}

func (e SolvingError) Report() *Report {
	return NewReport(e.rootPkg, e.cause)
}

func (e SolvingError) Error() string {
	return NewStandardTextReporter().Render(e.Report())
}
