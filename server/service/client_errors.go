package service

type SetupAlreadyErr interface {
	SetupAlready() bool
	Error() string
}

type setupAlreadyErr struct{}

func (e setupAlreadyErr) Error() string {
	return "Kolide Fleet has already been setup"
}

func (e setupAlreadyErr) SetupAlready() bool {
	return true
}

type InvalidLoginErr interface {
	InvalidLogin() bool
	Error() string
}

type invalidLoginErr struct{}

func (e invalidLoginErr) Error() string {
	return "The credentials supplied were invalid"
}

func (e invalidLoginErr) InvalidLogin() bool {
	return true
}

type NotSetupErr interface {
	NotSetup() bool
	Error() string
}

type notSetupErr struct{}

func (e notSetupErr) Error() string {
	return "The Kolide Fleet instance is not set up yet"
}

func (e notSetupErr) NotSetup() bool {
	return true
}

type NotFoundErr interface {
	NotFound() bool
	Error() string
}

type notFoundErr struct{}

func (e notFoundErr) Error() string {
	return "The resource was not found"
}

func (n notFoundErr) NotFound() bool {
	return true
}
