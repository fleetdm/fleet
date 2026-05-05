package fleet

import (
	"fmt"
	"regexp"
	"time"
)

type SecretVariable struct {
	Name      string    `json:"name" db:"name"`
	Value     string    `json:"value" db:"value"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

func (h SecretVariable) AuthzType() string {
	return "secret_variable"
}

// secretVariableFormat is regular expression to match only uppercase
// letters (A-Z), numbers (0-9), and underscores (_)
var secretVariableFormat = regexp.MustCompile(`^[A-Z0-9_]+$`)

// ValidateSecretVariableName validates the name of a secret variable.
func ValidateSecretVariableName(name string) error {
	const secretVariableMaxLen = 255
	if len(name) == 0 {
		return NewInvalidArgumentError("name", "secret variable name cannot be empty")
	}
	if len(name) > secretVariableMaxLen {
		return NewInvalidArgumentError("name", fmt.Sprintf("secret variable name is too long: %s", name))
	}
	if !secretVariableFormat.MatchString(name) {
		return NewInvalidArgumentError("name", fmt.Sprintf("secret variable with invalid format: %s", name))
	}
	return nil
}

// SecretVariableIdentifier holds identifier information about a secret variable (skipping the actual contents/value).
type SecretVariableIdentifier struct {
	ID        uint   `json:"id" db:"id"`
	Name      string `json:"name" name:"name"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}
