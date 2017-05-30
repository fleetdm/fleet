package kolide

// Datastore combines all the interfaces in the Kolide DAL
type Datastore interface {
	UserStore
	QueryStore
	CampaignStore
	PackStore
	LabelStore
	HostStore
	TargetStore
	PasswordResetStore
	SessionStore
	AppConfigStore
	InviteStore
	ScheduledQueryStore
	OptionStore
	DecoratorStore
	FileIntegrityMonitoringStore
	YARAStore
	LicenseStore
	Name() string
	Drop() error
	// MigrateTables creates and migrates the table schemas
	MigrateTables() error
	// MigrateData populates built-in data
	MigrateData() error
	// MigrationStatus returns nil if migrations are complete, and an error
	// if migrations need to be run.
	MigrationStatus() (MigrationStatus, error)
	Begin() (Transaction, error)
}

type MigrationStatus int

const (
	NoMigrationsCompleted = iota
	SomeMigrationsCompleted
	AllMigrationsCompleted
)

// NotFoundError is returned when the datastore resource cannot be found.
type NotFoundError interface {
	error
	IsNotFound() bool
}

// AlreadyExists is returned when creating a datastore resource that already
// exists.
type AlreadyExistsError interface {
	error
	IsExists() bool
}

type OptionalArg func() interface{}
