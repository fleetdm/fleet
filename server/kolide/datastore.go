package kolide

// Datastore combines all the interfaces in the Kolide DAL
type Datastore interface {
	UserStore
	QueryStore
	PackStore
	LabelStore
	HostStore
	PasswordResetStore
	SessionStore
	AppConfigStore
	InviteStore
	Name() string
	Drop() error
	Migrate() error
}
