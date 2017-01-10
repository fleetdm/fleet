package mock

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// implement kolide.NotFoundError
func (e *Error) IsNotFound() bool {
	return true
}

// implement kolide.AlreadyExistsError
func (e *Error) IsExists() bool {
	return true
}
