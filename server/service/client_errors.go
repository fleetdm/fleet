package service

type SetupAlreadyErr interface {
	SetupAlready() bool
	Error() string
}

type setupAlreadyErr struct {
	reason string
}

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

type invalidLoginErr struct {
	reason string
}

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

type notSetupErr struct {
	reason string
}

func (e notSetupErr) Error() string {
	return "The Kolide Fleet instance is not set up yet"
}

func (e notSetupErr) NotSetup() bool {
	return true
}
