package fleet

const (
	// ActionRead refers to reading an entity.
	ActionRead = "read"
	// ActionList refers to listing an entity.
	ActionList = "list"
	// ActionWrite refers to writing (CRUD operations) an entity.
	ActionWrite = "write"
	// ActionWriteRole is a write to a user's global roles and teams.
	ActionWriteRole = "write_role"
	// ActionWritePassword is a write to change a user's password.
	ActionWritePassword = "write_password"
	// ActionRun is the action for running a live query.
	ActionRun = "run"
	// ActionRunNew is the action for running a new live query.
	ActionRunNew = "run_new"
)
