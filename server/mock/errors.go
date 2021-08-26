package mock

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// IsNotFound implements fleet.NotFoundError
func (e *Error) IsNotFound() bool {
	return true
}

// IsExists implements fleet.AlreadyExistsError
func (e *Error) IsExists() bool {
	return true
}
