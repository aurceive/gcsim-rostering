package app

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	if e.Err == nil {
		return "exit"
	}
	return e.Err.Error()
}

func (e ExitError) Unwrap() error {
	return e.Err
}

func Exit(code int) error {
	return ExitError{Code: code}
}

func ExitWithError(code int, err error) error {
	return ExitError{Code: code, Err: err}
}

func asExitError(err error) (ExitError, bool) {
	if err == nil {
		return ExitError{}, false
	}
	ee, ok := err.(ExitError)
	return ee, ok
}
