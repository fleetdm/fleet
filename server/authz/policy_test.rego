package authz

team_user := {"teams": [
	{
		"id": 4,
		"role": "observer",
	},
	{
		"id": 5,
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

