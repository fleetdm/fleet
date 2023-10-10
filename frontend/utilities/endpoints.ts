const API_VERSION = "latest";

export default {
  ACTIVITIES: `/${API_VERSION}/fleet/activities`,
  CHANGE_PASSWORD: `/${API_VERSION}/fleet/change_password`,
  CONFIG: `/${API_VERSION}/fleet/config`,
  CONFIRM_EMAIL_CHANGE: (token: string): string => {
    return `/${API_VERSION}/fleet/email/change/${token}`;
  },
  DEVICE_USER_DETAILS: `/${API_VERSION}/fleet/device`,
  DEVICE_USER_MDM_ENROLLMENT_PROFILE: (token: string): string => {
    return `/${API_VERSION}/fleet/device/${token}/mdm/apple/manual_enrollment_profile`;
  },
  DEVICE_USER_RESET_ENCRYPTION_KEY: (token: string): string => {
    return `/${API_VERSION}/fleet/device/${token}/rotate_encryption_key`;
  },
  DOWNLOAD_INSTALLER: `/${API_VERSION}/fleet/download_installer`,
  ENABLE_USER: (id: number): string => {
    return `/${API_VERSION}/fleet/users/${id}/enable`;
  },
  FORGOT_PASSWORD: `/${API_VERSION}/fleet/forgot_password`,
  GLOBAL_ENROLL_SECRETS: `/${API_VERSION}/fleet/spec/enroll_secret`,
  GLOBAL_POLICIES: `/${API_VERSION}/fleet/policies`,
  GLOBAL_SCHEDULE: `/${API_VERSION}/fleet/schedule`,
  HOST_SUMMARY: `/${API_VERSION}/fleet/host_summary`,
  HOSTS: `/${API_VERSION}/fleet/hosts`,
  HOSTS_COUNT: `/${API_VERSION}/fleet/hosts/count`,
  HOSTS_DELETE: `/${API_VERSION}/fleet/hosts/delete`,
  HOSTS_REPORT: `/${API_VERSION}/fleet/hosts/report`,
  HOSTS_TRANSFER: `/${API_VERSION}/fleet/hosts/transfer`,
  HOSTS_TRANSFER_BY_FILTER: `/${API_VERSION}/fleet/hosts/transfer/filter`,
  INVITES: `/${API_VERSION}/fleet/invites`,
  LABELS: `/${API_VERSION}/fleet/labels`,
  LABEL_HOSTS: (id: number): string => {
    return `/${API_VERSION}/fleet/labels/${id}/hosts`;
  },
  LABEL_SPEC_BY_NAME: (labelName: string) => {
    return `/${API_VERSION}/fleet/spec/labels/${labelName}`;
  },
  LOGIN: `/${API_VERSION}/fleet/login`,
  LOGOUT: `/${API_VERSION}/fleet/logout`,
  MACADMINS: `/${API_VERSION}/fleet/macadmins`,

  // MDM endpoints
  MDM_APPLE: `/${API_VERSION}/fleet/mdm/apple`,
  MDM_APPLE_BM: `/${API_VERSION}/fleet/mdm/apple_bm`,
  MDM_APPLE_BM_KEYS: `/${API_VERSION}/fleet/mdm/apple/dep/key_pair`,
  MDM_SUMMARY: `/${API_VERSION}/fleet/hosts/summary/mdm`,
  MDM_REQUEST_CSR: `/${API_VERSION}/fleet/mdm/apple/request_csr`,
  MDM_PROFILES: `/${API_VERSION}/fleet/mdm/apple/profiles`,
  MDM_PROFILE: (id: number) => `/${API_VERSION}/fleet/mdm/apple/profiles/${id}`,
  MDM_UPDATE_APPLE_SETTINGS: `/${API_VERSION}/fleet/mdm/apple/settings`,
  MDM_PROFILES_AGGREGATE_STATUSES: `/${API_VERSION}/fleet/mdm/apple/profiles/summary`,
  MDM_DISK_ENCRYPTION_SUMMARY: `/${API_VERSION}/fleet/mdm/disk_encryption/summary`,
  MDM_APPLE_SSO: `/${API_VERSION}/fleet/mdm/sso`,
  MDM_APPLE_ENROLLMENT_PROFILE: (token: string, ref?: string) => {
    const query = new URLSearchParams({ token });
    if (ref) {
      query.append("enrollment_reference", ref);
    }
    return `/api/mdm/apple/enroll?${query}`;
  },
  MDM_BOOTSTRAP_PACKAGE_METADATA: (teamId: number) =>
    `/${API_VERSION}/fleet/mdm/apple/bootstrap/${teamId}/metadata`,
  MDM_BOOTSTRAP_PACKAGE: `/${API_VERSION}/fleet/mdm/apple/bootstrap`,
  MDM_BOOTSTRAP_PACKAGE_SUMMARY: `/${API_VERSION}/fleet/mdm/apple/bootstrap/summary`,
  MDM_SETUP: `/${API_VERSION}/fleet/mdm/apple/setup`,
  MDM_EULA: (token: string) =>
    `/${API_VERSION}/fleet/mdm/apple/setup/eula/${token}`,
  MDM_EULA_UPLOAD: `/${API_VERSION}/fleet/mdm/apple/setup/eula`,
  MDM_EULA_METADATA: `/${API_VERSION}/fleet/mdm/apple/setup/eula/metadata`,
  HOST_MDM: (id: number) => `/${API_VERSION}/fleet/hosts/${id}/mdm`,
  HOST_MDM_UNENROLL: (id: number) =>
    `/${API_VERSION}/fleet/mdm/hosts/${id}/unenroll`,
  HOST_ENCRYPTION_KEY: (id: number) =>
    `/${API_VERSION}/fleet/mdm/hosts/${id}/encryption_key`,

  ME: `/${API_VERSION}/fleet/me`,
  OS_VERSIONS: `/${API_VERSION}/fleet/os_versions`,
  OSQUERY_OPTIONS: `/${API_VERSION}/fleet/spec/osquery_options`,
  PACKS: `/${API_VERSION}/fleet/packs`,
  PERFORM_REQUIRED_PASSWORD_RESET: `/${API_VERSION}/fleet/perform_required_password_reset`,
  QUERIES: `/${API_VERSION}/fleet/queries`,
  RESET_PASSWORD: `/${API_VERSION}/fleet/reset_password`,
  RUN_QUERY: `/${API_VERSION}/fleet/queries/run`,
  SCHEDULE_QUERY: `/${API_VERSION}/fleet/packs/schedule`,
  SCHEDULED_QUERIES: (packId: number): string => {
    return `/${API_VERSION}/fleet/packs/${packId}/scheduled`;
  },
  SETUP: `/v1/setup`, // not a typo - hasn't been updated yet
  SOFTWARE: `/${API_VERSION}/fleet/software`,
  SSO: `/v1/fleet/sso`,
  STATUS_LABEL_COUNTS: `/${API_VERSION}/fleet/host_summary`,
  STATUS_LIVE_QUERY: `/${API_VERSION}/fleet/status/live_query`,
  STATUS_RESULT_STORE: `/${API_VERSION}/fleet/status/result_store`,
  TARGETS: `/${API_VERSION}/fleet/targets`,
  TEAM_POLICIES: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/policies`;
  },
  TEAM_SCHEDULE: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/schedule`;
  },
  TEAMS: `/${API_VERSION}/fleet/teams`,
  TEAMS_AGENT_OPTIONS: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/agent_options`;
  },
  TEAMS_ENROLL_SECRETS: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/secrets`;
  },
  TEAMS_MEMBERS: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/users`;
  },
  TEAMS_TRANSFER_HOSTS: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/hosts`;
  },
  UPDATE_USER_ADMIN: (id: number): string => {
    return `/${API_VERSION}/fleet/users/${id}/admin`;
  },
  USER_SESSIONS: (id: number): string => {
    return `/${API_VERSION}/fleet/users/${id}/sessions`;
  },
  USERS: `/${API_VERSION}/fleet/users`,
  USERS_ADMIN: `/${API_VERSION}/fleet/users/admin`,
  VERSION: `/${API_VERSION}/fleet/version`,

  // SCRIPTS
  HOST_SCRIPTS: (id: number) => `/${API_VERSION}/fleet/hosts/${id}/scripts`,
  SCRIPTS: `/${API_VERSION}/fleet/scripts`,
  SCRIPT: (id: number) => `/${API_VERSION}/fleet/scripts/${id}`,
  SCRIPT_RESULT: (executionId: string) =>
    `/${API_VERSION}/fleet/scripts/results/${executionId}`,
  SCRIPT_RUN: `/${API_VERSION}/fleet/scripts/run`,
};
