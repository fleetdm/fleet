package fleet

import (
	"fmt"
	"testing"
)

func TestValidateSecretVariableName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name:        "empty name",
			input:       "",
			expectedErr: NewInvalidArgumentError("name", "secret variable name cannot be empty"),
		},
		{
			name:        "name too long",
			input:       string(make([]byte, 256)),
			expectedErr: NewInvalidArgumentError("name", fmt.Sprintf("secret variable name is too long: %s", string(make([]byte, 256)))),
		},
		{
			name:        "invalid format - contains lowercase",
			input:       "lowercase_123",
			expectedErr: NewInvalidArgumentError("name", fmt.Sprintf("secret variable with invalid format: %s", "lowercase_123")),
		},
		{
			name:        "invalid format - special characters",
			input:       "invalid@name",
			expectedErr: NewInvalidArgumentError("name", fmt.Sprintf("secret variable with invalid format: %s", "invalid@name")),
		},
		{
			name:        "valid name",
			input:       "VALID123",
			expectedErr: nil,
		},
		{
			name:        "valid name with underscore",
			input:       "VALID_SECRET_NAME",
			expectedErr: nil,
		},
		{
			name:        "single character",
			input:       "A",
			expectedErr: nil,
		},
		{
			name:        "valid format - starts with number",
			input:       "1_",
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecretVariableName(tt.input)
			if err != nil && tt.expectedErr != nil {
				if err.Error() != tt.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else if err != tt.expectedErr {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}
