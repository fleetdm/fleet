package inmem

import "fmt"

type notFoundError struct {
	ID           uint
	Message      string
	ResourceType string
}

func notFound(kind string) *notFoundError {
	return &notFoundError{
		ResourceType: kind,
	}
}

func (e *notFoundError) Error() string {
	if e.ID == 0 {
		return fmt.Sprintf("%s was not found in the datastore", e.ResourceType)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s, %s was not found in the datastore", e.ResourceType, e.Message)
	}
	return fmt.Sprintf("%s %d was not found in the datastore", e.ResourceType, e.ID)
}

func (e *notFoundError) WithID(id uint) error {
	e.ID = id
	return e
}

func (e *notFoundError) WithMessage(msg string) error {
	e.Message = msg
	return e
}

func (e *notFoundError) IsNotFound() bool {
	return true
}

type existsError struct {
	ID           uint
	ResourceType string
}

func alreadyExists(kind string, id uint) error {
	return &existsError{
		ID:           id,
		ResourceType: kind,
	}
}

func (e *existsError) Error() string {
	return fmt.Sprintf("%s %d already exists in the datastore", e.ResourceType, e.ID)
}

func (e *existsError) IsExists() bool {
	return true
}
