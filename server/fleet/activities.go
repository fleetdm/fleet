package fleet

import (
	"context"
	"encoding/json"
)

const (
	// ActivityTypeCreatedPack is the activity type for created packs
	ActivityTypeCreatedPack = "created_pack"
	// ActivityTypeEditedPack is the activity type for edited packs
	ActivityTypeEditedPack = "edited_pack"
	// ActivityTypeDeletedPack is the activity type for deleted packs
	ActivityTypeDeletedPack = "deleted_pack"
	// ActivityTypeAppliedSpecPack is the activity type for pack specs applied
	ActivityTypeAppliedSpecPack = "applied_spec_pack"
	// ActivityTypeCreatedSavedQuery is the activity type for created saved queries
	ActivityTypeCreatedSavedQuery = "created_saved_query"
	// ActivityTypeEditedSavedQuery is the activity type for edited saved queries
	ActivityTypeEditedSavedQuery = "edited_saved_query"
	// ActivityTypeDeletedSavedQuery is the activity type for deleted saved queries
	ActivityTypeDeletedSavedQuery = "deleted_saved_query"
	// ActivityTypeDeletedMultipleSavedQuery is the activity type for multiple deleted saved queries
	ActivityTypeDeletedMultipleSavedQuery = "deleted_multiple_saved_query"
	// ActivityTypeAppliedSpecSavedQuery is the activity type for saved queries spec applied
	ActivityTypeAppliedSpecSavedQuery = "applied_spec_saved_query"
	// ActivityTypeCreatedTeam is the activity type for created team
	ActivityTypeCreatedTeam = "created_team"
	// ActivityTypeDeletedTeam is the activity type for deleted team
	ActivityTypeDeletedTeam = "deleted_team"
	// ActivityTypeLiveQuery is the activity type for live queries
	ActivityTypeLiveQuery = "live_query"
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
	ActorID       *uint            `json:"actor_id" db:"user_id"`
	ActorGravatar *string          `json:"actor_gravatar" db:"gravatar_url"`
	ActorEmail    *string          `json:"actor_email" db:"email"`
	Type          string           `json:"type" db:"activity_type"`
	Details       *json.RawMessage `json:"details" db:"details"`
}

// AuthzType implement AuthzTyper to be able to verify access to activities
func (*Activity) AuthzType() string {
	return "activity"
}
