package spec

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
)

// Validate checks the rendered document against the OpenAPI 3.1 schema.
func Validate(specBytes []byte) error {
	doc, err := libopenapi.NewDocument(specBytes)
	if err != nil {
		return fmt.Errorf("parsing generated spec: %w", err)
	}
	v, errs := validator.NewValidator(doc)
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	ok, verrs := v.ValidateDocument()
	if !ok {
		var msgs []string
		for _, e := range verrs {
			msgs = append(msgs, e.Message)
		}
		return fmt.Errorf("document is not valid OpenAPI 3.1:\n  %s", strings.Join(msgs, "\n  "))
	}
	return nil
}
