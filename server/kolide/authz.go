package kolide

const (
	// ActionRead refers to reading an entity.
	ActionRead = "read"
	// ActionWrite refers to writing (CRUD operations) an entity.
	ActionWrite = "write"
	// ActionWriteRole is a write to a user's global roles and teams.
	ActionWriteRole = "write_role"
	// ActionRun is the action for running a live query.
	ActionRun = "run"
)
