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
