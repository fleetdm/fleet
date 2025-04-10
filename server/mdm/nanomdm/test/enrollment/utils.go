package enrollment

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// HTTPError contains the body and status details.
type HTTPError struct {
	Body       []byte
	Status     string
	StatusCode int
}

func NewHTTPError(response *http.Response, body []byte) *HTTPError {
	if response == nil {
		response = &http.Response{}
	}
	return &HTTPError{
		Body:       body,
		Status:     response.Status,
		StatusCode: response.StatusCode,
	}
}

// Error returns strings for HTTP errors that may include body and status.
func (e *HTTPError) Error() (err string) {
	err = "HTTP error"
	if e == nil {
		return
	}
	if e.Status != "" {
		err += ": " + e.Status
	} else {
		err += ": " + strconv.Itoa(e.StatusCode)
	}
	if len(e.Body) > 0 {
		err += ": " + string(e.Body)
	}
	return
}

const Limit10KiB = 10 * 1024

// HTTPErrors reports an HTTP error for a non-200 HTTP response.
// The first 10KiB of the body is read for non-200 response.
// For a 200 response nil is returned.
// Caller is responsible for closing response body.
func HTTPErrors(r *http.Response) error {
	if r == nil {
		return errors.New("nil response")
	}

	if r.StatusCode != 200 {
		body, err := io.ReadAll(io.LimitReader(r.Body, Limit10KiB))
		if err != nil {
			return fmt.Errorf("error reading body of non-200 response: %w", err)
		}
		return NewHTTPError(r, body)
	}

	return nil
}
