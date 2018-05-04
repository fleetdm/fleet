package service

type SetupAlreadyErr interface {
	SetupAlready() bool
	Error() string
}

type setupAlreadyErr struct {
	reason string
}

func (e setupAlreadyErr) Error() string {
	return e.reason
}

func (e setupAlreadyErr) SetupAlready() bool {
	return true
}

func setupAlready() error {
	return setupAlreadyErr{
		reason: "Kolide Fleet has already been setup",
	}
}

type InvalidLoginErr interface {
	InvalidLogin() bool
	Error() string
}

type invalidLoginErr struct {
	reason string
}

func (e invalidLoginErr) Error() string {
	return e.reason
}

func (e invalidLoginErr) InvalidLogin() bool {
	return true
}

func invalidLogin() error {
	return invalidLoginErr{
		reason: "The credentials supplied were invalid",
	}
}

type NotSetupErr interface {
	NotSetup() bool
	Error() string
}

type notSetupErr struct {
	reason string
}

func (e notSetupErr) Error() string {
	return e.reason
}

func (e notSetupErr) NotSetup() bool {
	return true
}

func notSetup() error {
	return notSetupErr{
		reason: "The Kolide Fleet instance is not setup yet",
	}
}
