package fleet

type SecretVariable struct {
	Name  string `json:"name" db:"name"`
	Value string `json:"value" db:"value"`
}

func (h SecretVariable) AuthzType() string {
	return "secret_variable"
}
