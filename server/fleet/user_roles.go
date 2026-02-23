package fleet

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
