package pubgrub

type SolvingError struct {
	cause *Incompatibility
}

func (e SolvingError) Cause() *Incompatibility {
	return e.cause
}

func (e SolvingError) Error() string {
	return GetErrorMessage(e.cause)
}
