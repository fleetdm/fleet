# See OPA documentation for specification of this policy language:
# https://www.openpolicyagent.org/docs/latest/policy-language/

package authz

import input.action
import input.object
import input.subject

# Constants

# Actions
read := "read"
list := "list"
write := "write"
write_role := "write_role"
run := "run"
run_new := "run_new"

# Roles
admin := "admin"
maintainer := "maintainer"
observer := "observer"

# Default deny
default allow = false

# team_role gets the role that the subject has for the team, returning undefined
# if the user has no explicit role for that team.
team_role(subject, team_id) = role {
	subject_team := subject.teams[_]
	subject_team.id == team_id
	role := subject_team.role
}

##
# Global config
##

# Any logged in user can read global config
allow {
  object.type == "app_config"
  not is_null(subject)
  action == read
}

# Admin can write global config
allow {
  object.type == "app_config"
  subject.global_role == admin
  action == write
}

##
# Teams
##

# Any logged in user can read teams (service must filter appropriately based on
# access).
allow {
  object.type == "team"
  not is_null(subject)
  action == read
}

# Admin can write teams
allow {
  object.type == "team"
  subject.global_role == admin
  action == write
}

##
# Users
##

# Any user can write self (besides role)
allow {
  object.type == "user"
  object.id == subject.id
	action == write
}

# Any user can read other users
allow {
  object.type == "user"
  not is_null(subject)
  action == read
}

# Admins can write all users + roles
allow {
  object.type == "user"
  subject.global_role == admin
	action == [write, write_role][_]
}

##
# Invites
##

# Only global admins may read/write invites
allow {
  object.type == "invite"
  subject.global_role == admin
  action == read
}
allow {
  object.type == "invite"
  subject.global_role == admin
  action == write
}

##
# Activities
##

# All users can read activities
allow {
  not is_null(subject)
  object.type == "activity"
  action == read
}

##
# Sessions
##

# Any user can read/write own session
allow {
  object.type == "session"
  object.user_id == subject.id
	action == [read, write][_]
}

# Admins can read/write all user sessions
allow {
  object.type == "session"
  subject.global_role == admin
	action == [read, write][_]
}

##
# Enroll Secrets
##

# Admins can read/write all
allow {
	object.type == "enroll_secret"
	subject.global_role == admin
  action == [read, write][_]
}

# Global maintainers can read all
allow {
	object.type == "enroll_secret"
	subject.global_role == maintainer
	action == read
}

# Team maintainers can read for appropriate teams
allow {
	object.type == "enroll_secret"
	team_role(subject, object.team_id) == maintainer
	action == read
}

# (Observers are not granted read for enroll secrets)

##
# Hosts
##

# Allow anyone to list (must be filtered appropriately by the service).
allow {
  object.type == "host"
  not is_null(subject)
  action == list
}

# Allow read/write for global admin/maintainer
allow {
	object.type == "host"
	subject.global_role = admin
	action == [read, write][_]
}
allow {
	object.type == "host"
	subject.global_role = maintainer
	action == [read, write][_]
}

# Allow read for global observer
allow {
	object.type == "host"
	subject.global_role = observer
	action == read
}

# Allow read for matching team maintainer/observer
allow {
	object.type == "host"
	team_role(subject, object.team_id) == maintainer
	action == read
}
allow {
	object.type == "host"
	team_role(subject, object.team_id) == observer
	action == read
}

# Team maintainers can write to hosts of their own team
allow {
	object.type == "host"
	team_role(subject, object.team_id) == maintainer
	action == write
}

##
# Labels
##

# All users can read labels
allow {
  object.type == "label"
  not is_null(subject)
  action == read
}

# Only global admins and maintainers can write labels
allow {
  object.type == "label"
  subject.global_role == admin
  action == write
}
allow {
  object.type == "label"
  subject.global_role == maintainer
  action == write
}

##
# Queries
##

# All users can read queries
allow {
  not is_null(subject)
  object.type == "query"
  action == read
}

# Only admins and maintainers can write queries
allow {
  object.type == "query"
  subject.global_role == admin
  action == write
}
allow {
  object.type == "query"
  subject.global_role == maintainer
  action == write
}

# Team maintainers can create new queries
allow {
  object.id == 0 # new queries have ID zero
  object.type == "query"
  team_role(subject, subject.teams[_].id) == maintainer
  action == write
}

# Team maintainers can edit and delete only their own queries
allow {
  object.author_id == subject.id
  object.type == "query"
  team_role(subject, subject.teams[_].id) == maintainer
  action == write
}

# Global admins and (team) maintainers can run any
allow {
  object.type == "query"
  subject.global_role == admin
  action = run
}
allow {
  object.type == "query"
  subject.global_role == maintainer
  action = run
}
allow {
  object.type == "query"
  subject.global_role == admin
  action = run_new
}
allow {
  object.type == "query"
  subject.global_role == maintainer
  action = run_new
}
# Team maintainer running a non-observers_can_run query must have the targets
# filtered to only teams that they maintain
allow {
  object.type == "query"
  # If role is maintainer on any team
  team_role(subject, subject.teams[_].id) == maintainer
  action == run
}

# Team maintainer can run a new query
allow {
  object.type == "query"
  # If role is maintainer on any team
  team_role(subject, subject.teams[_].id) == maintainer
  action == run_new
}

# Team admin can run a new query
allow {
  object.type == "query"
  # If role is maintainer on any team
  team_role(subject, subject.teams[_].id) == admin
  action == run_new
}

# (Team) observers can run only if observers_can_run
allow {
	object.type == "query"
	object.observer_can_run == true
	subject.global_role == observer
	action = run
}
# Team observer running a observers_can_run query must have the targets
# filtered to only teams that they observe
allow {
	object.type == "query"
	object.observer_can_run == true
	# If role is observer on any team
	team_role(subject, subject.teams[_].id) == observer
	action == run
}

##
# Targets
##

# All users can read targets (filtered appropriately based on their
# teams/roles).
allow {
  not is_null(subject)
  object.type == "target"
  action == read
}

##
# Packs
##

# Global admins and maintainers and team maintainers can read/write packs
allow {
  object.type == "pack"
  subject.global_role == admin
  action == [read, write][_]
}
allow {
  object.type == "pack"
  subject.global_role == maintainer
  action == [read, write][_]
}

# Team maintainers can read global packs
allow {
  is_null(object.team_ids)
  object.type == "pack"
  team_role(subject, subject.teams[_].id) == maintainer
  action == read
}

allow {
  object.team_ids[_] == subject.teams[_].id
  object.type == "pack"
  team_role(subject, subject.teams[_].id) == maintainer
  action == [read, write][_]
}

##
# File Carves
##

# Only global admins can read/write carves
allow {
  object.type == "carve"
  subject.global_role == admin
  action == [read, write][_]
}

##
# Policies
##

# Global Admin and Maintainer users can read and write policies
allow {
  object.type == ["policy","team_policy"][_]
  subject.global_role == admin
  action == [read, write][_]
}

allow {
  is_null(object.team_id)
  object.type == "policy"
  subject.global_role == maintainer
  action == [read, write][_]
}

allow {
  object.type == "policy"
  subject.global_role == maintainer
  action == [read][_]
}

# Global Observer users can read policies
allow {
  object.type == "policy"
  subject.global_role == observer
  action == [read][_]
}

# Team Maintainers can read and write policies
allow {
  not is_null(object.team_id)
  object.team_id == subject.teams[_].id
  object.type == "policy"
  team_role(subject, subject.teams[_].id) == maintainer
  action == [read, write][_]
}

# Team maintainers can read global policies

allow {
  is_null(object.team_id)
  object.type == "policy"
  team_role(subject, subject.teams[_].id) == maintainer
  action == read
}

# Team Observer can read policies
allow {
  not is_null(object.team_id)
  object.team_id == subject.teams[_].id
  object.type == "policy"
  team_role(subject, subject.teams[_].id) == observer
  action == [read][_]
}

##
# Software
##

# All users can read software
allow {
  not is_null(subject)
  object.type == "software"
  action == read
}