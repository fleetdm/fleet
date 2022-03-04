package service

type badRequestError struct {
	message string
}

func (e *badRequestError) Error() string {
	return e.message
}

func (e *badRequestError) BadRequestError() []map[string]string {
	return nil
}

type alreadyExistsError struct{}

func (a alreadyExistsError) Error() string {
	return "Entity already exists"
}

func (a alreadyExistsError) IsExists() bool {
	return true
}
