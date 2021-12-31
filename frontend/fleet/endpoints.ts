import { IPack } from "interfaces/pack";
import { IScheduledQuery } from "interfaces/scheduled_query";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";

export default {
  ACTIVITIES: "/v1/fleet/activities",
  CHANGE_PASSWORD: "/v1/fleet/change_password",
  CONFIG: "/v1/fleet/config",
  CONFIRM_EMAIL_CHANGE: (token: string): string => {
    return `/v1/fleet/email/change/${token}`;
  },
  ENABLE_USER: (id: number): string => {
    return `/v1/fleet/users/${id}/enable`;
  },
  FORGOT_PASSWORD: "/v1/fleet/forgot_password",
  GLOBAL_ENROLL_SECRETS: "/v1/fleet/spec/enroll_secret",
  GLOBAL_POLICIES: "/v1/fleet/global/policies",
  GLOBAL_SCHEDULE: "/v1/fleet/global/schedule",
  HOST_SUMMARY: (teamId: number | undefined): string => {
    const teamString = teamId ? `?team_id=${teamId}` : "";
    return `/v1/fleet/host_summary${teamString}`;
  },
  HOSTS: "/v1/fleet/hosts",
  HOSTS_COUNT: "/v1/fleet/hosts/count",
  HOSTS_DELETE: "/v1/fleet/hosts/delete",
  HOSTS_TRANSFER: "/v1/fleet/hosts/transfer",
  HOSTS_TRANSFER_BY_FILTER: "/v1/fleet/hosts/transfer/filter",
  INVITES: "/v1/fleet/invites",
  LABELS: "/v1/fleet/labels",
  LABEL_HOSTS: (id: number): string => {
    return `/v1/fleet/labels/${id}/hosts`;
  },
  LOGIN: "/v1/fleet/login",
  LOGOUT: "/v1/fleet/logout",
  ME: "/v1/fleet/me",
  OSQUERY_OPTIONS: "/v1/fleet/spec/osquery_options",
  PACKS: "/v1/fleet/packs",
  PERFORM_REQUIRED_PASSWORD_RESET: "/v1/fleet/perform_required_password_reset",
  QUERIES: "/v1/fleet/queries",
  RESET_PASSWORD: "/v1/fleet/reset_password",
  RUN_QUERY: "/v1/fleet/queries/run",
  SCHEDULED_QUERIES: "/v1/fleet/schedule",
  SCHEDULED_QUERY: (id: number): string => {
    return `/v1/fleet/packs/${id}/scheduled`;
  },
  SETUP: "/v1/setup",
  SOFTWARE: "/v1/fleet/software",
  SSO: "/v1/fleet/sso",
  STATUS_LABEL_COUNTS: "/v1/fleet/host_summary",
  STATUS_LIVE_QUERY: "/v1/fleet/status/live_query",
  STATUS_RESULT_STORE: "/v1/fleet/status/result_store",
  TARGETS: "/v1/fleet/targets",
  TEAM_POLICIES: (teamId: number): string => {
    return `/v1/fleet/teams/${teamId}/policies`;
  },
  TEAM_SCHEDULE: (teamId: number): string => {
    return `/v1/fleet/teams/${teamId}/schedule`;
  },
  TEAMS: "/v1/fleet/teams",
  TEAMS_AGENT_OPTIONS: (teamId: number): string => {
    return `/v1/fleet/teams/${teamId}/agent_options`;
  },
  TEAMS_ENROLL_SECRETS: (teamId: number): string => {
    return `/v1/fleet/teams/${teamId}/secrets`;
  },
  TEAMS_MEMBERS: (teamId: number): string => {
    return `/v1/fleet/teams/${teamId}/users`;
  },
  TEAMS_TRANSFER_HOSTS: (teamId: number): string => {
    return `/v1/fleet/teams/${teamId}/hosts`;
  },
  UPDATE_USER_ADMIN: (id: number): string => {
    return `/v1/fleet/users/${id}/admin`;
  },
  USER_SESSIONS: (id: number): string => {
    return `/v1/fleet/users/${id}/sessions`;
  },
  USERS: "/v1/fleet/users",
  USERS_ADMIN: "/v1/fleet/users/admin",
  VERSION: "/v1/fleet/version",
};
