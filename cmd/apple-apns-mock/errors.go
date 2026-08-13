package main

import "net/http"

type Statuser interface {
	Status() int
}

type apnsError struct {
	reason string
	status int
}

func (e *apnsError) Error() string { return e.reason }
func (e *apnsError) Status() int   { return e.status }

func newAPNSError(reason string, status int) *apnsError {
	return &apnsError{reason: reason, status: status}
}

func BadRequestError() *apnsError {
	return newAPNSError("BadRequest", http.StatusBadRequest)
}

func InvalidPushTypeError() *apnsError {
	return newAPNSError("InvalidPushType", http.StatusBadRequest)
}

func BadDeviceTokenError() *apnsError {
	return newAPNSError("BadDeviceToken", http.StatusBadRequest)
}

func MissingDeviceTokenError() *apnsError {
	return newAPNSError("MissingDeviceToken", http.StatusBadRequest)
}

func BadExpirationDateError() *apnsError {
	return newAPNSError("BadExpirationDate", http.StatusBadRequest)
}

func PayloadEmptyError() *apnsError {
	return newAPNSError("PayloadEmpty", http.StatusBadRequest)
}

func BadMessageIdError() *apnsError {
	return newAPNSError("BadMessageId", http.StatusBadRequest)
}

func PayloadTooLargeError() *apnsError {
	return newAPNSError("PayloadTooLarge", http.StatusRequestEntityTooLarge)
}

func InternalServerError() *apnsError {
	return newAPNSError("InternalServerError", http.StatusInternalServerError)
}
