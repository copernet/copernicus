package util

type ProjectError struct {
	ErrorCode int
	Description string
}

func (e ProjectError)Error() string {
	return e.Description
}

func IsErrorCode(err error, c int) bool {
	e, ok := err.(ProjectError)
	return ok && e.ErrorCode == c
}

func ErrToProject(errorCode int, reason string) error {
	return ProjectError{ErrorCode:errorCode, Description:reason}
}
