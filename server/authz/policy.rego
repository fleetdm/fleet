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

# Action used on object "targeted_query" used for running live queries.
run := "run"
# Action used on object "query" used for running "new" live queries.
run_new := "run_new"

# Roles
admin := "admin"
maintainer := "maintainer"
observer := "observer"
observer_plus := "observer_plus"
gitops := "gitops"

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

# Global admin, maintainer, observer_plus and observer can read global config.
allow {
  object.type == "app_config"
  subject.global_role == [admin, maintainer, observer_plus, observer][_]
  action == read
}

# Team admin, maintainer, observer_plus and observer can read global config.
allow {
  object.type == "app_config"
  # If role is admin, maintainer, observer_plus or observer on any team.
  team_role(subject, subject.teams[_].id) == [admin, maintainer, observer_plus, observer][_]
  action == read
}

# Global admins and gitops can write global config.
allow {
  object.type == "app_config"
  subject.global_role == [admin, gitops][_]
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

# Global admins, maintainers, observer_plus and observers can read teams.
allow {
  object.type == "team"
  object.id != 0
  subject.global_role == [admin, maintainer, observer, observer_plus][_]
  action == read
}

# Team admins, maintainers, observer_plus and observers can read their team.
allow {
  object.type == "team"
  object.id != 0
  team_role(subject, object.id) == [admin, maintainer, observer, observer_plus][_]
  action == read
}

# Global admins and gitops can write teams.
allow {
  object.type == "team"
  subject.global_role == [admin, gitops][_]
  action == write
}

# Team admins and gitops can write their teams.
allow {
  object.type == "team"
  team_role(subject, object.id) == [admin, gitops][_]
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

# Global admins, maintainers, observer_plus and observers can read activities.
allow {
  object.type == "activity"
  subject.global_role == [admin, maintainer, observer_plus, observer][_]
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

# Global admins and maintainers can read/write enroll secrets.
allow {
	object.type == "enroll_secret"
	subject.global_role == [admin, maintainer][_]
  action == [read, write][_]
}

# Global gitops can write global enroll secrets.
allow {
  object.type == "enroll_secret"
  object.is_global_secret
  subject.global_role == gitops
  action == write
}

# Team admins and maintainers can read/write for appropriate teams.
allow {
	object.type == "enroll_secret"
	team_role(subject, object.team_id) == [admin, maintainer][_]
	action == [read, write][_]
}

# (Observers are not granted read for enroll secrets)

##
# Hosts
##

# Global admins, maintainers, observer_plus and observers can list hosts.
allow {
  object.type == "host"
  subject.global_role == [admin, maintainer, observer_plus, observer][_]
  action == list
}

# Team admins, maintainers, observer_plus and observers can list hosts.
allow {
	object.type == "host"
  # If role is admin, maintainer, observer_plus or observer on any team.
  team_role(subject, subject.teams[_].id) == [admin, maintainer, observer_plus, observer][_]
	action == list
}

# Allow read/write for global admin/maintainer.
allow {
	object.type == "host"
  subject.global_role == [admin, maintainer][_]
	action == [read, write][_]
}

# Global gitops can write hosts.
allow {
	object.type == "host"
  subject.global_role == gitops
	action == write
}

# Allow read for global observer and observer_plus.
allow {
	object.type == "host"
	subject.global_role == [observer, observer_plus][_]
	action == read
}

# Allow read for matching team admin/maintainer/observer/observer_plus.
allow {
	object.type == "host"
	team_role(subject, object.team_id) == [admin, maintainer, observer, observer_plus][_]
	action == read
}

# Team admins and maintainers can write to hosts of their own team
allow {
	object.type == "host"
	team_role(subject, object.team_id) == [admin, maintainer][_]
	action == write
}

##
# Labels
##

# Global admins, maintainers, observer_plus and observers can read labels.
allow {
  object.type == "label"
	subject.global_role == [admin, maintainer, observer_plus, observer][_]
  action == read
}

# Team admins, maintainers, observer_plus and observers can read labels.
allow {
	object.type == "label"
  # If role is admin, maintainer, observer_plus or observer on any team.
  team_role(subject, subject.teams[_].id) == [admin, maintainer, observer_plus, observer][_]
	action == read
}

# Only global admins, maintainers and gitops can write labels
allow {
  object.type == "label"
  subject.global_role == [admin, maintainer, gitops][_]
  action == write
}

##
# Queries
##

# Global admins, maintainers and gitops can write queries.
allow {
  object.type == "query"
  subject.global_role == [admin, maintainer, gitops][_]
  action == write
}

# Global admins, maintainers, observer_plus and observers can read queries.
allow {
  object.type == "query"
  subject.global_role == [admin, maintainer, observer_plus, observer][_]
  action == read
}

# Team admin, maintainers and gitops can write queries for their teams.
allow {
  object.type == "query"
  not is_null(object.team_id)
  team_role(subject, object.team_id) == [admin, maintainer, gitops][_]
  action == write
}

# Team admins, maintainers, observer_plus and observers can read queries for their teams.
allow {
  object.type == "query"
  not is_null(object.team_id)
  team_role(subject, object.team_id) == [admin, maintainer, observer_plus, observer][_]
  action == read
}

# Team admins, maintainers, observer_plus and observers can read global queries.
allow {
  object.type == "query"
  is_null(object.team_id)
  team_role(subject, subject.teams[_].id) == [admin, maintainer, observer_plus, observer][_]
  action == read
}

# Global admins, maintainers and observer_plus can run any query saved query.
allow {
  object.type == "targeted_query"
  subject.global_role == [admin, maintainer, observer_plus][_]
  action = run
}

# Global admins, maintainers and observer_plus can run any new query.
allow {
  object.type == "query"
  subject.global_role == [admin, maintainer, observer_plus][_]
  action = run_new
}

# Team admin, maintainer and observer_plus running a global non-observers_can_run query
# must have the targets filtered to only teams that they maintain.
allow {
  object.type == "targeted_query"
  object.observer_can_run == false
  is_null(subject.global_role)
  action == run

  is_null(object.team_id)

  not is_null(object.host_targets.teams)
  ok_teams := { tmid | tmid := object.host_targets.teams[_]; team_role(subject, tmid) == [admin, maintainer, observer_plus][_] }
  count(ok_teams) == count(object.host_targets.teams)
}

# Team admin, maintainer and observer_plus running a non-observers_can_run query that belongs to their team
# must have the targets filtered to only teams that they maintain.
allow {
  object.type == "targeted_query"
  object.observer_can_run == false
  is_null(subject.global_role)
  action == run

  team_role(subject, object.team_id) == [admin, maintainer, observer_plus][_]

  not is_null(object.host_targets.teams)
  ok_teams := { tmid | tmid := object.host_targets.teams[_]; team_role(subject, tmid) == [admin, maintainer, observer_plus][_] }
  count(ok_teams) == count(object.host_targets.teams)
}

# Team admin, maintainer and observer_plus running a global non-observers_can_run query when no target teams are specified.
allow {
  object.type == "targeted_query"
  object.observer_can_run == false
  is_null(subject.global_role)
  action == run

  is_null(object.team_id)

  # If role is admin, maintainer or observer_plus on any team.
  team_role(subject, subject.teams[_].id) == [admin, maintainer, observer_plus][_]

  # and there are no team targets
  is_null(object.host_targets.teams)
}

# Team admin, maintainer and observer_plus running a non-observers_can_run query that belongs to their team when no target teams are specified.
allow {
  object.type == "targeted_query"
  object.observer_can_run == false
  is_null(subject.global_role)
  action == run

  team_role(subject, object.team_id) == [admin, maintainer, observer_plus][_]

  # there are no team targets
  is_null(object.host_targets.teams)
}

# Team admin, maintainer and observer_plus can run a new query.
allow {
  object.type == "query"
  # If role is admin, maintainer or observer_plus on any team.
  team_role(subject, subject.teams[_].id) == [admin, maintainer, observer_plus][_]
  action == run_new
}

# Global observers can run only if observers_can_run.
allow {
  object.type == "targeted_query"
  object.observer_can_run == true
  subject.global_role == observer
  action = run
}

# Team admin, maintainer, observer_plus and observer running a global observers_can_run query must have the targets
# filtered to only teams that they observe.
allow {
  object.type == "targeted_query"
  object.observer_can_run == true
  is_null(subject.global_role)
  action == run

  is_null(object.team_id)

  not is_null(object.host_targets.teams)
  ok_teams := { tmid | tmid := object.host_targets.teams[_]; team_role(subject, tmid) == [admin, maintainer, observer_plus, observer][_] }
  count(ok_teams) == count(object.host_targets.teams)
}

# Team admin, maintainer, observer_plus and observer running an observers_can_run query that belongs to their team must have the targets
# filtered to only teams that they observe.
allow {
  object.type == "targeted_query"
  object.observer_can_run == true
  is_null(subject.global_role)
  action == run

  team_role(subject, object.team_id) == [admin, maintainer, observer_plus, observer][_]

  not is_null(object.host_targets.teams)
  ok_teams := { tmid | tmid := object.host_targets.teams[_]; team_role(subject, tmid) == [admin, maintainer, observer_plus, observer][_] }
  count(ok_teams) == count(object.host_targets.teams)
}

# Team admin, maintainer, observer_plus and observer running a global observers_can_run query and there are no target teams.
allow {
  object.type == "targeted_query"
  object.observer_can_run == true
  is_null(subject.global_role)
  action == run

  is_null(object.team_id)

  # If role is admin, maintainer, observer_plus or observer on any team.
  team_role(subject, subject.teams[_].id) == [admin, maintainer, observer_plus, observer][_]

  # and there are no team targets
  is_null(object.host_targets.teams)
}

# Team admin, maintainer, observer_plus and observer running an observers_can_run query that belongs to their team and there are no target teams.
allow {
  object.type == "targeted_query"
  object.observer_can_run == true
  is_null(subject.global_role)
  action == run

  team_role(subject, object.team_id) == [admin, maintainer, observer_plus, observer][_]

  # there are no team targets
  is_null(object.host_targets.teams)
}

##
# Targets
##

# Global admin, maintainer, observer_plus and observer can read targets.
allow {
  object.type == "target"
  subject.global_role == [admin, maintainer, observer_plus, observer][_]
  action == read
}

# Team admin, maintainer, observer_plus and observer can read global config.
allow {
  object.type == "target"
  # If role is admin, maintainer, observer_plus or observer on any team.
  team_role(subject, subject.teams[_].id) == [admin, maintainer, observer_plus, observer][_]
  action == read
}

##
# 2017 Packs (deprecated)
##

# Global admins, maintainers and gitops can read/write 2017 packs.
allow {
  object.type == "pack"
  subject.global_role == [admin, maintainer, gitops][_]
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

# Global admins and maintainers can read and write policies.
allow {
  object.type == "policy"
  subject.global_role == [admin, maintainer][_]
  action == [read, write][_]
}

# Global gitops can write policies.
allow {
  object.type == "policy"
  subject.global_role == gitops
  action == write
}

# Global observer and observer_plus can read any policies.
allow {
  object.type == "policy"
  subject.global_role == [observer, observer_plus][_]
  action == read
}

# Team admin and maintainers can read and write policies for their teams.
allow {
  not is_null(object.team_id)
  object.type == "policy"
  team_role(subject, object.team_id) == [admin, maintainer][_]
  action == [read, write][_]
}

# Team gitops can write policies for their teams.
allow {
  not is_null(object.team_id)
  object.type == "policy"
  team_role(subject, object.team_id) == gitops
  action == write
}

# Team admin, maintainers, observers and observers_plus can read global policies
allow {
  is_null(object.team_id)
  object.type == "policy"
  team_role(subject, subject.teams[_].id) == [admin, maintainer, observer, observer_plus][_]
  action == read
}

# Team observer and observer_plus can read policies for their teams.
allow {
  not is_null(object.team_id)
  object.type == "policy"
  team_role(subject, object.team_id) == [observer, observer_plus][_]
  action == read
}

##
# Software
##

# Global admins, maintainers, observers and observer_plus can read all software.
allow {
  object.type == "software_inventory"
  subject.global_role == [admin, maintainer, observer, observer_plus][_]
  action == read
}

# Team admins, maintainers, observers and observer_plus can read all software in their teams.
allow {
  not is_null(object.team_id)
  object.type == "software_inventory"
  team_role(subject, object.team_id) == [admin, maintainer, observer, observer_plus][_]
  action == read
}

##
# Apple MDM
##

# Global admins and maintainers can read and write Apple MDM config profiles.
allow {
  object.type == "mdm_apple_config_profile"
  subject.global_role == [admin, maintainer][_]
  action == [read, write][_]
}

# Global gitops can write Apple MDM config profiles.
allow {
  object.type == "mdm_apple_config_profile"
  subject.global_role == gitops
  action == write
}

# Team admins and maintainers can read and write Apple MDM config profiles on their teams.
allow {
  not is_null(object.team_id)
  object.team_id != 0
  object.type == "mdm_apple_config_profile"
  team_role(subject, object.team_id) == [admin, maintainer][_]
  action == [read, write][_]
}

# Team gitops can write Apple MDM config profiles on their teams.
allow {
  not is_null(object.team_id)
  object.team_id != 0
  object.type == "mdm_apple_config_profile"
  team_role(subject, object.team_id) == gitops
  action == write
}

# Global admins can read and write MDM apple information.
allow {
  object.type == "mdm_apple"
  subject.global_role == admin
  action == [read, write][_]
}

# Global admins can read and write Apple MDM enrollments.
allow {
  object.type == "mdm_apple_enrollment_profile"
  subject.global_role == admin
  action == [read, write][_]
}

# Global admins and maintainers can write (execute) MDM Apple commands.
allow {
  object.type == "mdm_apple_command"
  subject.global_role == [admin, maintainer][_]
  action == write
}

# Team admins and maintainers can write (execute) MDM Apple commands on hosts of their teams.
allow {
  not is_null(object.team_id)
  object.type == "mdm_apple_command"
  team_role(subject, object.team_id) == [admin, maintainer][_]
  action == write
}

# Global admins, maintainers, observers and observer_plus can read MDM Apple commands.
allow {
  object.type == "mdm_apple_command"
  subject.global_role == [admin, maintainer, observer, observer_plus][_]
  action == read
}

# Team admins, maintainers, observers and observer_plus can read MDM Apple commands on hosts of their teams.
allow {
  not is_null(object.team_id)
  object.type == "mdm_apple_command"
  team_role(subject, object.team_id) == [admin, maintainer, observer, observer_plus][_]
  action == read
}

# Global admins can read and write Apple MDM installers.
allow {
  object.type == "mdm_apple_installer"
  subject.global_role == admin
  action == [read, write][_]
}

# Global admins can read and write Apple devices.
allow {
  object.type == "mdm_apple_device"
  subject.global_role == admin
  action == [read, write][_]
}

# Global admins can read and write Apple DEP devices.
allow {
  object.type == "mdm_apple_dep_device"
  subject.global_role == admin
  action == [read, write][_]
}

# Global admins and maintainers can read and write MDM Apple settings.
allow {
  object.type == "mdm_apple_settings"
  subject.global_role == [admin, maintainer][_]
  action == [read, write][_]
}

# Global gitops can write MDM Apple settings.
allow {
  object.type == "mdm_apple_settings"
  subject.global_role == gitops
  action == write
}

# Team admins and maintainers can read and write MDM Apple Settings of their teams.
allow {
  not is_null(object.team_id)
  object.type == "mdm_apple_settings"
  team_role(subject, object.team_id) == [admin, maintainer][_]
  action == [read, write][_]
}

# Team gitops can write MDM Apple Settings of their teams.
allow {
  not is_null(object.team_id)
  object.type == "mdm_apple_settings"
  team_role(subject, object.team_id) == gitops
  action == write
}

# Global admins and maintainers can read and write bootstrap packages.
allow {
  object.type == "mdm_apple_bootstrap_package"
  subject.global_role == [admin, maintainer][_]
  action == [read, write][_]
}

# Global gitops can write bootstrap packages.
allow {
  object.type == "mdm_apple_bootstrap_package"
  subject.global_role == gitops
  action == write
}

# Team admins and maintainers can read and write bootstrap packages on their teams.
allow {
  not is_null(object.team_id)
  object.team_id != 0
  object.type == "mdm_apple_bootstrap_package"
  team_role(subject, object.team_id) == [admin, maintainer][_]
  action == [read, write][_]
}

# Team gitops can write bootstrap packages on their teams.
allow {
  not is_null(object.team_id)
  object.team_id != 0
  object.type == "mdm_apple_bootstrap_package"
  team_role(subject, object.team_id) == gitops
  action == write
}

##
# MDM Apple Setup Assistant
##

# Global admins and maintainers can read and write macos setup assistants.
allow {
  object.type == "mdm_apple_setup_assistant"
  subject.global_role == [admin, maintainer][_]
  action == [read, write][_]
}

# Global gitops can write macos setup assistants.
allow {
  object.type == "mdm_apple_setup_assistant"
  subject.global_role == gitops
  action == write
}

# Team admins and maintainers can read and write macos setup assistants on their teams.
allow {
  not is_null(object.team_id)
  object.team_id != 0
  object.type == "mdm_apple_setup_assistant"
  team_role(subject, object.team_id) == [admin, maintainer][_]
  action == [read, write][_]
}

# Team gitops can write macos setup assistants on their teams.
allow {
  not is_null(object.team_id)
  object.team_id != 0
  object.type == "mdm_apple_setup_assistant"
  team_role(subject, object.team_id) == gitops
  action == write
}

##
# Cron schedules
##

# Global admins can read and write (i.e. trigger) cron schedules.
allow {
  object.type == "cron_schedules"
  subject.global_role == admin
  action == [read, write][_]
}

##
# Version
##

# Any logged in user can read Fleet's version
allow {
  object.type == "version"
  not is_null(subject)
  action == read
}

##
# Host Script Result (script execution and output)
##

# Global admins and maintainers can write (execute) anonymous scripts (not
# gitops as this is not something that relates to fleetctl apply).
allow {
  object.type == "host_script_result"
  is_null(object.script_id)
  subject.global_role == [admin, maintainer][_]
  action == write
}

# Global admins, maintainers, observer_plus and observers can write (execute)
# saved scripts (not gitops as this is not something that relates to fleetctl
# apply).
allow {
  object.type == "host_script_result"
  not is_null(object.script_id)
  subject.global_role == [admin, maintainer, observer, observer_plus][_]
  action == write
}

# Global admins, maintainers, observer_plus and observers can read scripts.
allow {
  object.type == "host_script_result"
  subject.global_role == [admin, maintainer, observer, observer_plus][_]
  action == read
}

# Team admin and maintainers can write (execute) anonymous scripts for their
# teams (not gitops as this is not something that relates to fleetctl apply).
allow {
  object.type == "host_script_result"
  is_null(object.script_id)
  not is_null(object.team_id)
  team_role(subject, object.team_id) == [admin, maintainer][_]
  action == write
}

# Team admins, maintainers, observer_plus and observers can write (execute)
# saved scripts for their teams (not gitops as this is not something that
# relates to fleetctl apply).
allow {
  object.type == "host_script_result"
  not is_null(object.script_id)
  not is_null(object.team_id)
  team_role(subject, object.team_id) == [admin, maintainer, observer_plus, observer][_]
  action == write
}

# Team admins, maintainers, observer_plus and observers can read scripts for their teams.
allow {
  object.type == "host_script_result"
  not is_null(object.team_id)
  team_role(subject, object.team_id) == [admin, maintainer, observer_plus, observer][_]
  action == read
}

##
# Scripts (saved script)
##

# Global admins and maintainers can write (upload) saved scripts.
allow {
  object.type == "script"
  subject.global_role == [admin, maintainer][_]
  action == write
}

# Global admins, maintainers, observer_plus and observers can read scripts.
allow {
  object.type == "script"
  subject.global_role == [admin, maintainer, observer, observer_plus][_]
  action == read
}

# Team admin and maintainers can write (upload) saved scripts for their teams.
allow {
  object.type == "script"
  not is_null(object.team_id)
  team_role(subject, object.team_id) == [admin, maintainer][_]
  action == write
}

# Team admins, maintainers, observer_plus and observers can read scripts for their teams.
allow {
  object.type == "script"
  not is_null(object.team_id)
  team_role(subject, object.team_id) == [admin, maintainer, observer_plus, observer][_]
  action == read
}
