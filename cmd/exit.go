package cmd

type ExitCodeError struct {
	Code int
	Err  error
}

func (e ExitCodeError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "command failed"
}

func (e ExitCodeError) Unwrap() error {
	return e.Err
}
