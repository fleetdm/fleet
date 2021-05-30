package authz

team_user := {"teams": [
	{
		"team_id": 4,
		"role": "observer",
	},
	{
		"team_id": 5,
		"role": "maintainer",
	},
]}

global_admin := {
	"global_role": "admin",
	"teams": [],
}

global_maintainer := {
	"global_role": "maintainer",
	"teams": [],
}

global_observer := {
	"global_role": "observer",
	"teams": [],
}

enroll_secret_team_4 := {
	"type": "enroll_secret",
	"team_id": 4,
}

test_team_role {
  team_role(team_user, 4) == "observer"
  team_role(team_user, 5) == "maintainer" 
  not team_role(team_user, 2)
}

test_enroll_secret_global_admin {
	allow with input.subject as global_admin
		 with input.object as {"type": "enroll_secret", "team": 4}
		 with input.action as "read"

	allow with input.subject as global_admin
		 with input.object as {"type": "enroll_secret", "team": 4}
		 with input.action as "write"
}

test_enroll_secret_global_maintainer {
	allow with input.subject as global_maintainer
		 with input.object as {"type": "enroll_secret", "team": 4}
		 with input.action as "read"

	not allow with input.subject as global_maintainer
		 with input.object as {"type": "enroll_secret", "team": 4}
		 with input.action as "write"
}

test_enroll_secret_global_observer {
	not allow with input.subject as global_observer
		 with input.object as {"type": "enroll_secret", "team": 4}
		 with input.action as "read"

	not allow with input.subject as global_observer
		 with input.object as {"type": "enroll_secret", "team": 4}
		 with input.action as "write"
}

test_enroll_secret_team_user {
	# Allows read for team where user is maintainer
	allow with input.subject as team_user
		 with input.object as {"type": "enroll_secret", "team": 5}
		 with input.action as "read"

	not allow with input.subject as global_observer
		 with input.object as {"type": "enroll_secret", "team": 4}
		 with input.action as "write"

	# Denies for team where user is observer
	not allow with input.subject as team_user
		 with input.object as {"type": "enroll_secret", "team": 4}
		 with input.action as "read"

	not allow with input.subject as global_observer
		 with input.object as {"type": "enroll_secret", "team": 4}
		 with input.action as "write"

	# Denies for team where user has no role
	not allow with input.subject as team_user
		 with input.object as {"type": "enroll_secret", "team": 2}
		 with input.action as "read"

	not allow with input.subject as global_observer
		 with input.object as {"type": "enroll_secret", "team": 2}
		 with input.action as "write"
}
