package mock

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// implement fleet.NotFoundError
func (e *Error) IsNotFound() bool {
	return true
}

// implement fleet.AlreadyExistsError
func (e *Error) IsExists() bool {
	return true
}
