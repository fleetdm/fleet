package seed

import (
	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/themes"
)

// SeedUsers creates `count` users on the Fleet server using names drawn from
// the given theme. Roles cycle through observer / observer_plus / maintainer /
// admin / gitops so the seeded set covers every permission level.
//
// All users share a known dev password so tests can sign in as them; production
// Fleets should never run this against a real deployment.
const SeededUserPassword = "DibbleSeed123!"

var seededRoles = []string{
	"observer", "observer_plus", "maintainer", "admin", "gitops",
}

func Users(c Client, log Logger, theme themes.Theme, count int) Result {
	res := Result{Entity: "users"}
	for i := 0; i < count; i++ {
		name := themes.FullName(theme, i)
		email := themes.Email(theme, i)
		role := seededRoles[i%len(seededRoles)]
		body := map[string]any{
			"name":                        name,
			"email":                       email,
			"password":                    SeededUserPassword,
			"global_role":                 role,
			"admin_forced_password_reset": false,
		}
		err := c.Post("/api/latest/fleet/users/admin", body, nil)
		switch {
		case err == nil:
			res.Created++
			log.Printf("user %s <%s> [%s]", name, email, role)
		case IsAlreadyExists(err):
			res.Skipped++
		default:
			res.Errors = append(res.Errors, err)
		}
	}
	return res
}
