import { IPack } from "interfaces/pack";
import { IScheduledQuery } from "interfaces/scheduled_query";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";

export default {
  CHANGE_PASSWORD: "/v1/fleet/change_password",
  CONFIG: "/v1/fleet/config",
  VERSION: "/v1/fleet/version",
  CONFIRM_EMAIL_CHANGE: (token: string): string => {
    return `/v1/fleet/email/change/${token}`;
  },
  OSQUERY_OPTIONS: "/v1/fleet/spec/osquery_options",
  ENABLE_USER: (id: number): string => {
    return `/v1/fleet/users/${id}/enable`;
  },
  FORGOT_PASSWORD: "/v1/fleet/forgot_password",
  ACTIVITIES: "/v1/fleet/activities",
  GLOBAL_SCHEDULE: "/v1/fleet/global/schedule",
  HOSTS: "/v1/fleet/hosts",
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
  PACKS: "/v1/fleet/packs",
  PERFORM_REQUIRED_PASSWORD_RESET: "/v1/fleet/perform_required_password_reset",
  QUERIES: "/v1/fleet/queries",
  RESET_PASSWORD: "/v1/fleet/reset_password",
  RUN_QUERY: "/v1/fleet/queries/run",
  SCHEDULED_QUERIES: "/v1/fleet/schedule",
  SCHEDULED_QUERY: (pack: IPack): string => {
    return `/v1/fleet/packs/${pack.id}/scheduled`;
  },
  SETUP: "/v1/setup",
  STATUS_LABEL_COUNTS: "/v1/fleet/host_summary",
  TARGETS: "/v1/fleet/targets",
  TEAM_SCHEDULE: (id: number): string => {
    return `/v1/fleet/team/${id}/schedule`;
  },
  USERS: "/v1/fleet/users",
  USERS_ADMIN: "/v1/fleet/users/admin",
  UPDATE_USER_ADMIN: (id: number): string => {
    return `/v1/fleet/users/${id}/admin`;
  },
  USER_SESSIONS: (id: number): string => {
    return `/v1/fleet/users/${id}/sessions`;
  },
  SSO: "/v1/fleet/sso",
  STATUS_LIVE_QUERY: "/v1/fleet/status/live_query",
  STATUS_RESULT_STORE: "/v1/fleet/status/result_store",
  TEAMS: "/v1/fleet/teams",
  TEAMS_MEMBERS: (teamId: number): string => {
    return `/v1/fleet/teams/${teamId}/users`;
  },
  TEAMS_TRANSFER_HOSTS: (teamId: number): string => {
    return `/v1/fleet/teams/${teamId}/hosts`;
  },
  TEAMS_ENROLL_SECRETS: (teamId: number): string => {
    return `/v1/fleet/teams/${teamId}/secrets`;
  },
  TEAMS_AGENT_OPTIONS: (teamId: number): string => {
    return `/v1/fleet/teams/${teamId}/agent_options`;
  },
};
