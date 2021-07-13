package fleet

import (
	"context"
	"encoding/json"
)

const (
	ActivityTypeCreatedPack               = "created_pack"
	ActivityTypeEditedPack                = "edited_pack"
	ActivityTypeDeletedPack               = "deleted_pack"
	ActivityTypeAppliedSpecPack           = "applied_spec_pack"
	ActivityTypeCreatedSavedQuery         = "created_saved_query"
	ActivityTypeEditedSavedQuery          = "edited_saved_query"
	ActivityTypeDeletedSavedQuery         = "deleted_saved_query"
	ActivityTypeDeletedMultipleSavedQuery = "deleted_multiple_saved_query"
	ActivityTypeAppliedSpecSavedQuery     = "applied_spec_saved_query"
	ActivityTypeCreatedTeam               = "created_team"
	ActivityTypeDeletedTeam               = "deleted_team"
	ActivityTypeLiveQuery                 = "live_query"
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

func (*Activity) AuthzType() string {
	return "activity"
}
