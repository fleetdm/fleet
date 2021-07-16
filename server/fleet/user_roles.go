package fleet

import "context"

const (
	UserRolesKind = "user_roles"
)

type UsersRoleSpec struct {
	Roles map[string]*UserRoleSpec `json:"roles"`
}

type UserRoleSpec struct {
	GlobalRole *string        `json:"global_role"`
	Teams      []TeamRoleSpec `json:"teams"`
}

type TeamRoleSpec struct {
	Name string `json:"team"`
	Role string `json:"role"`
}

type UserRolesService interface {
	// ApplyUserRolesSpecs applies a list of user global and team role changes
	ApplyUserRolesSpecs(ctx context.Context, specs UsersRoleSpec) error
}
