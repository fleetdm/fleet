package fleet

import (
	"context"
	"encoding/json"
)

const (
	CreatedPackActivityType               = "created_pack"
	EditedPackActivityType                = "edited_pack"
	DeletedPackActivityType               = "deleted_pack"
	AppliedSpecPackActivityType           = "applied_spec_pack"
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
	ListActivities(opt ListOptions) ([]*Activity, error)
}

type ActivitiesService interface {
	ListActivities(ctx context.Context, opt ListOptions) ([]*Activity, error)
}

type Activity struct {
	CreateTimestamp
	ID            uint             `json:"id" db:"id"`
	ActorFullName string           `json:"actor_full_name" db:"name"`
	ActorId       uint             `json:"actor_id" db:"user_id"`
	Type          string           `json:"type" db:"activity_type"`
	Details       *json.RawMessage `json:"details" db:"details"`
}
