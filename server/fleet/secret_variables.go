package fleet

import "time"

type SecretVariable struct {
	Name      string    `json:"name" db:"name"`
	Value     string    `json:"value" db:"value"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

func (h SecretVariable) AuthzType() string {
	return "secret_variable"
}
