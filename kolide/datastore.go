package kolide

// Datastore combines all the interfaces in the Kolide DAL
type Datastore interface {
	UserStore
	QueryStore
	PackStore
	OsqueryStore
	HostStore
	EmailStore
	SessionStore
	Name() string
	Drop() error
	Migrate() error
}
