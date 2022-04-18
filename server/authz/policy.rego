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

# User specific actions
write_role := "write_role"
change_password := "change_password"

# Query specific actions
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
# access) if the overall object is specified
allow {
  object.type == "team"
  object.id == 0
  not is_null(subject)
  action == read
}
# For specific teams, only members can read
allow {
  object.type == "team"
  object.id != 0
  team_role(subject, object.id) == [admin,maintainer][_]
  action == read
}
# or global admins or global maintainers
allow {
  object.type == "team"
  object.id != 0
  subject.global_role == [admin, maintainer][_]
  action == read
}

# Admin can write teams
allow {
  object.type == "team"
  subject.global_role == admin
  action == write
}

# Team admin can write teams
allow {
  object.type == "team"
  team_role(subject, object.id) == admin
  action == write
}

##
# Users
#
# NOTE: More rules apply to users but they are implemented in Go code.
# Our end goal is to move all the authorization logic here.
##

# Any user can read and write self and change their own password.
allow {
  object.type == "user"
  object.id == subject.id
  object.id != 0
  action == [read, write, change_password][_]
}

# Global admins can perform all operations on all users.
allow {
  object.type == "user"
  subject.global_role == admin
  action == [read, write, write_role, change_password][_]
}

# Team admins can perform all operations on the team users (except changing their password).
allow {
  object.type == "user"
  team_role(subject, object.teams[_].id) == admin
  action == [read, write, write_role][_]
}

##
# Invites
##

# Global admins may read/write invites
allow {
  object.type == "invite"
  subject.global_role == admin
  action == [read,write][_]
}

##
# Activities
##

# Only global users can read activities
allow {
  not is_null(subject.global_role)
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

# Global admins and maintainers can read/write all
allow {
	object.type == "enroll_secret"
	subject.global_role == [admin, maintainer][_]
  action == [read, write][_]
}

# Team admins and maintainers can read/write for appropriate teams
allow {
	object.type == "enroll_secret"
	team_role(subject, object.team_id) == [admin, maintainer][_]
	action == [read, write][_]
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

# Allow read for matching team admin/maintainer/observer
allow {
	object.type == "host"
	team_role(subject, object.team_id) == [admin, maintainer, observer][_]
	action == read
}

# Team admins and maintainers can write to hosts of their own team
allow {
	object.type == "host"
	team_role(subject, object.team_id) == [admin,maintainer][_]
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

# Global admins and maintainers can write queries
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

# Team admins and maintainers can create new queries
allow {
  object.id == 0 # new queries have ID zero
  object.type == "query"
  team_role(subject, subject.teams[_].id) == [admin, maintainer][_]
  action == write
}

# Team admins and maintainers can edit and delete only their own queries
allow {
  object.author_id == subject.id
  object.type == "query"
  team_role(subject, subject.teams[_].id) == [admin,maintainer][_]
  action == write
}

# Global admins and maintainers can run any
allow {
  object.type == "targeted_query"
  subject.global_role == admin
  action = run
}
allow {
  object.type == "targeted_query"
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

# Team admin and maintainer running a non-observers_can_run query must have the targets
# filtered to only teams that they maintain.
allow {
  object.type == "targeted_query"
  object.observer_can_run == false
  is_null(subject.global_role)
  action == run

  not is_null(object.host_targets.teams)
  ok_teams := { tmid | tmid := object.host_targets.teams[_]; team_role(subject, tmid) == [admin,maintainer][_] }
  count(ok_teams) == count(object.host_targets.teams)
}

# Team admin and maintainer running a non-observers_can_run query when no target teams
# are specified.
allow {
  object.type == "targeted_query"
  object.observer_can_run == false
  is_null(subject.global_role)
  action == run

  # If role is admin or maintainer on any team
  team_role(subject, subject.teams[_].id) == [admin,maintainer][_]

  # and there are no team targets
  is_null(object.host_targets.teams)
}

# Team admin and maintainer can run a new query
allow {
  object.type == "query"
  # If role is admin or maintainer on any team
  team_role(subject, subject.teams[_].id) == [admin,maintainer][_]
  action == run_new
}

# Observers can run only if observers_can_run
allow {
  object.type == "targeted_query"
  object.observer_can_run == true
  subject.global_role == observer
  action = run
}

# Team observer running a observers_can_run query must have the targets
# filtered to only teams that they observe.
allow {
  object.type == "targeted_query"
  object.observer_can_run == true
  is_null(subject.global_role)
  action == run

  not is_null(object.host_targets.teams)
  ok_teams := { tmid | tmid := object.host_targets.teams[_]; team_role(subject, tmid) == [admin,maintainer,observer][_] }
  count(ok_teams) == count(object.host_targets.teams)
}

# Team observer running a observers_can_run query and there are no
# target teams.
allow {
  object.type == "targeted_query"
  object.observer_can_run == true
  is_null(subject.global_role)
  action == run

  # If role is admin, maintainer or observer on any team
  team_role(subject, subject.teams[_].id) == [admin,maintainer,observer][_]

  # and there are no team targets
  is_null(object.host_targets.teams)
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

# Global admins and maintainers can read/write all packs.
allow {
  object.type == "pack"
  subject.global_role == [admin, maintainer][_]
  action == [read, write][_]
}

# All users can read the global pack.
allow {
  object.type == "pack"
  not is_null(subject)
  object.is_global_pack == true
  action == read
}

# Team admins, maintainers and observers can read their team's pack.
#
# NOTE: Action "read" on a team's pack includes listing its scheduled queries.
allow {
  object.type == "pack"
  not is_null(object.pack_team_id)
  team_role(subject, object.pack_team_id) == [admin, maintainer, observer][_]
  action == read
}

# Team admins and maintainers can add/remove scheduled queries from/to their team's pack.
#
# NOTE: The team's pack is not editable per-se, it's a special pack to group
# all the team's scheduled queries. So the "write" operation only covers
# adding/removing scheduled queries from the pack.
allow {
  object.type == "pack"
  not is_null(object.pack_team_id)
  team_role(subject, object.pack_team_id) == [admin, maintainer][_]
  action == write
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

# Global Admin and Maintainer can read and write policies
allow {
  object.type == "policy"
  subject.global_role == [admin,maintainer][_]
  action == [read, write][_]
}

# Global Observer can read any policies
allow {
  object.type == "policy"
  subject.global_role == observer
  action == read
}

# Team admin and maintainers can read and write policies for their teams
allow {
  not is_null(object.team_id)
  object.type == "policy"
  team_role(subject, object.team_id) == [admin,maintainer][_]
  action == [read, write][_]
}

# Team admin, maintainers and observers can read global policies
allow {
  is_null(object.team_id)
  object.type == "policy"
  team_role(subject, subject.teams[_].id) == [admin,maintainer,observer][_]
  action == read
}

# Team Observer can read policies for their teams
allow {
  not is_null(object.team_id)
  object.type == "policy"
  team_role(subject, object.team_id) == observer
  action == read
}

##
# Software
##

# Global users can read all software.
allow {
  object.type == "software_inventory"
  subject.global_role == [admin, maintainer, observer][_]
  action == read
}

# Team users can read all software in their teams.
allow {
  not is_null(object.team_id)
  object.type == "software_inventory"
  team_role(subject, object.team_id) == [admin, maintainer, observer][_]
  action == read
}