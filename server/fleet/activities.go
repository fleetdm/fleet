package fleet

const (
	CreatedPackActivityType               = "created_pack"
	EditedPackActivityType                = "edited_pack"
	DeletedPackActivityType               = "deleted_pack"
	CreatedSavedQueryActivityType         = "created_saved_query"
	EditedSavedQueryActivityType          = "edited_saved_query"
	DeletedSavedQueryActivityType         = "deleted_saved_query"
	DeletedMultipleSavedQueryActivityType = "deleted_multiple_saved_query"
	AppliedSpecSavedQueryActivityType     = "applied_spec_saved_query"
	CreatedTeamActivityType               = "created_team"
	DeletedTeamActivityType               = "deleted_team"
	LiveQueryActivityType                 = "live_query"
)

type ActivitiesStore interface {
	NewActivity(user *User, activityType string, details *map[string]interface{}) error
}
