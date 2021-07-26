package fleet

// Datastore combines all the interfaces in the Fleet DAL
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
	CarveStore
	TeamStore
	SoftwareStore
	ActivitiesStore
	StatisticsStore

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

func IsNotFound(err error) bool {
	e, ok := err.(NotFoundError)
	if !ok {
		return false
	}
	return e.IsNotFound()
}

// AlreadyExists is returned when creating a datastore resource that already
// exists.
type AlreadyExistsError interface {
	error
	IsExists() bool
}

// ForeignKeyError is returned when the operation fails due to foreign key
// constraints.
type ForeignKeyError interface {
	error
	IsForeignKey() bool
}

func IsForeignKey(err error) bool {
	e, ok := err.(ForeignKeyError)
	if !ok {
		return false
	}
	return e.IsForeignKey()
}

type OptionalArg func() interface{}
