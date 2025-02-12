package common

// TODO: Move the authz.go from fleet package to here.
// Why? Because the fleet package has many dependencies while this package has no other fleet package dependencies.
const (
	// ActionRead refers to reading an entity.
	ActionRead = "read"
	// ActionList refers to listing an entity.
	ActionList = "list"
	// ActionWrite refers to writing (CRUD operations) an entity.
	ActionWrite = "write"
)
