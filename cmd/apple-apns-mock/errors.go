package main

import "net/http"

type Statuser interface {
	Status() int
}

type BadRequestError struct {
	Reason string
}

func (e *BadRequestError) Error() string {
	return e.Reason
}

func (e *BadRequestError) Status() int {
	return http.StatusBadRequest
}

type invalidPushTypeError struct {
	BadRequestError
}

func InvalidPushTypeError() *invalidPushTypeError {
	return &invalidPushTypeError{
		BadRequestError: BadRequestError{
			Reason: "InvalidPushType",
		},
	}
}

type badDeviceTokenError struct {
	BadRequestError
}

func BadDeviceTokenError() *badDeviceTokenError {
	return &badDeviceTokenError{
		BadRequestError: BadRequestError{
			Reason: "BadDeviceToken",
		},
	}
}

type missingDeviceTokenError struct {
	BadRequestError
}

func MissingDeviceTokenError() *missingDeviceTokenError {
	return &missingDeviceTokenError{
		BadRequestError: BadRequestError{
			Reason: "MissingDeviceToken",
		},
	}
}

type badExpirationDateError struct {
	BadRequestError
}

func BadExpirationDateError() *badExpirationDateError {
	return &badExpirationDateError{
		BadRequestError: BadRequestError{
			Reason: "BadExpirationDate",
		},
	}
}

type payloadEmptyError struct {
	BadRequestError
}

func PayloadEmptyError() *payloadEmptyError {
	return &payloadEmptyError{
		BadRequestError: BadRequestError{
			Reason: "PayloadEmpty",
		},
	}
}

type badMessageIdError struct {
	BadRequestError
}

func BadMessageIdError() *badMessageIdError {
	return &badMessageIdError{
		BadRequestError: BadRequestError{
			Reason: "BadMessageId",
		},
	}
}

type payloadTooLargeError struct {
	reason string
}

func (e *payloadTooLargeError) Error() string {
	return e.reason
}

func (e *payloadTooLargeError) Status() int {
	return http.StatusRequestEntityTooLarge
}

func PayloadTooLargeError() *payloadTooLargeError {
	return &payloadTooLargeError{
		reason: "PayloadTooLarge",
	}
}

type internalServerError struct {
	reason string
}

func (e *internalServerError) Error() string {
	return e.reason
}

func (e *internalServerError) Status() int {
	return http.StatusInternalServerError
}

func InternalServerError() *internalServerError {
	return &internalServerError{
		reason: "InternalServerError",
	}
}
