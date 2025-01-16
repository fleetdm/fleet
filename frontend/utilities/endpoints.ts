const API_VERSION = "latest";

export default {
  // activities
  ACTIVITIES: `/${API_VERSION}/fleet/activities`,
  HOST_PAST_ACTIVITIES: (id: number): string => {
    return `/${API_VERSION}/fleet/hosts/${id}/activities`;
  },
  HOST_UPCOMING_ACTIVITIES: (id: number): string => {
    return `/${API_VERSION}/fleet/hosts/${id}/activities/upcoming`;
  },

  CHANGE_PASSWORD: `/${API_VERSION}/fleet/change_password`,
  CONFIG: `/${API_VERSION}/fleet/config`,
  CONFIRM_EMAIL_CHANGE: (token: string): string => {
    return `/${API_VERSION}/fleet/email/change/${token}`;
  },

  DOWNLOAD_INSTALLER: `/${API_VERSION}/fleet/download_installer`,
  ENABLE_USER: (id: number): string => {
    return `/${API_VERSION}/fleet/users/${id}/enable`;
  },
  FORGOT_PASSWORD: `/${API_VERSION}/fleet/forgot_password`,
  GLOBAL_ENROLL_SECRETS: `/${API_VERSION}/fleet/spec/enroll_secret`,
  GLOBAL_POLICIES: `/${API_VERSION}/fleet/policies`,
  GLOBAL_SCHEDULE: `/${API_VERSION}/fleet/schedule`,

  // Device endpoints
  DEVICE_USER_DETAILS: `/${API_VERSION}/fleet/device`,
  DEVICE_SOFTWARE: (token: string) =>
    `/${API_VERSION}/fleet/device/${token}/software`,
  DEVICE_SOFTWARE_INSTALL: (token: string, softwareTitleId: number) =>
    `/${API_VERSION}/fleet/device/${token}/software/install/${softwareTitleId}`,
  DEVICE_USER_MDM_ENROLLMENT_PROFILE: (token: string): string => {
    return `/${API_VERSION}/fleet/device/${token}/mdm/apple/manual_enrollment_profile`;
  },
  DEVICE_TRIGGER_LINUX_DISK_ENCRYPTION_KEY_ESCROW: (token: string): string => {
    return `/${API_VERSION}/fleet/device/${token}/mdm/linux/trigger_escrow`;
  },

  // Host endpoints
  HOST_SUMMARY: `/${API_VERSION}/fleet/host_summary`,
  HOST_QUERY_REPORT: (hostId: number, queryId: number) =>
    `/${API_VERSION}/fleet/hosts/${hostId}/queries/${queryId}`,
  HOSTS: `/${API_VERSION}/fleet/hosts`,
  HOSTS_COUNT: `/${API_VERSION}/fleet/hosts/count`,
  HOSTS_DELETE: `/${API_VERSION}/fleet/hosts/delete`,
  HOSTS_REPORT: `/${API_VERSION}/fleet/hosts/report`,
  HOSTS_TRANSFER: `/${API_VERSION}/fleet/hosts/transfer`,
  HOSTS_TRANSFER_BY_FILTER: `/${API_VERSION}/fleet/hosts/transfer/filter`,
  HOST_LOCK: (id: number) => `/${API_VERSION}/fleet/hosts/${id}/lock`,
  HOST_UNLOCK: (id: number) => `/${API_VERSION}/fleet/hosts/${id}/unlock`,
  HOST_WIPE: (id: number) => `/${API_VERSION}/fleet/hosts/${id}/wipe`,
  HOST_RESEND_PROFILE: (hostId: number, profileUUID: string) =>
    `/${API_VERSION}/fleet/hosts/${hostId}/configuration_profiles/${profileUUID}/resend`,
  HOST_SOFTWARE: (id: number) => `/${API_VERSION}/fleet/hosts/${id}/software`,
  HOST_SOFTWARE_PACKAGE_INSTALL: (hostId: number, softwareId: number) =>
    `/${API_VERSION}/fleet/hosts/${hostId}/software/${softwareId}/install`,
  HOST_SOFTWARE_PACKAGE_UNINSTALL: (hostId: number, softwareId: number) =>
    `/${API_VERSION}/fleet/hosts/${hostId}/software/${softwareId}/uninstall`,

  INVITES: `/${API_VERSION}/fleet/invites`,
  INVITE_VERIFY: (token: string) => `/${API_VERSION}/fleet/invites/${token}`,

  // labels
  LABEL: (id: number) => `/${API_VERSION}/fleet/labels/${id}`,
  LABELS: `/${API_VERSION}/fleet/labels`,
  LABELS_SUMMARY: `/${API_VERSION}/fleet/labels/summary`,
  LABEL_HOSTS: (id: number): string => {
    return `/${API_VERSION}/fleet/labels/${id}/hosts`;
  },
  LABEL_SPEC_BY_NAME: (labelName: string) => {
    return `/${API_VERSION}/fleet/spec/labels/${labelName}`;
  },

  LOGIN: `/${API_VERSION}/fleet/login`,
  CREATE_SESSION: `/${API_VERSION}/fleet/sessions`,
  LOGOUT: `/${API_VERSION}/fleet/logout`,
  MACADMINS: `/${API_VERSION}/fleet/macadmins`,

  /**
   * MDM endpoints
   */

  MDM_SUMMARY: `/${API_VERSION}/fleet/hosts/summary/mdm`,

  // apple mdm endpoints
  MDM_APPLE: `/${API_VERSION}/fleet/mdm/apple`,

  // Apple Business Manager (ABM) endpoints
  MDM_ABM_TOKENS: `/${API_VERSION}/fleet/abm_tokens`,
  MDM_ABM_TOKEN: (id: number) => `/${API_VERSION}/fleet/abm_tokens/${id}`,
  MDM_ABM_TOKEN_RENEW: (id: number) =>
    `/${API_VERSION}/fleet/abm_tokens/${id}/renew`,
  MDM_ABM_TOKEN_TEAMS: (id: number) =>
    `/${API_VERSION}/fleet/abm_tokens/${id}/teams`,
  MDM_APPLE_ABM_PUBLIC_KEY: `/${API_VERSION}/fleet/mdm/apple/abm_public_key`,
  MDM_APPLE_APNS_CERTIFICATE: `/${API_VERSION}/fleet/mdm/apple/apns_certificate`,
  MDM_APPLE_PNS: `/${API_VERSION}/fleet/apns`,
  MDM_APPLE_BM: `/${API_VERSION}/fleet/abm`, // TODO: Deprecated?
  MDM_APPLE_BM_KEYS: `/${API_VERSION}/fleet/mdm/apple/dep/key_pair`,
  MDM_APPLE_VPP_APPS: `/${API_VERSION}/fleet/software/app_store_apps`,
  MDM_REQUEST_CSR: `/${API_VERSION}/fleet/mdm/apple/request_csr`,

  // Apple VPP endpoints
  MDM_APPLE_VPP_TOKEN: `/${API_VERSION}/fleet/mdm/apple/vpp_token`, // TODO: Deprecated?
  MDM_VPP_TOKENS: `/${API_VERSION}/fleet/vpp_tokens`,
  MDM_VPP_TOKEN: (id: number) => `/${API_VERSION}/fleet/vpp_tokens/${id}`,
  MDM_VPP_TOKENS_RENEW: (id: number) =>
    `/${API_VERSION}/fleet/vpp_tokens/${id}/renew`,
  MDM_VPP_TOKEN_TEAMS: (id: number) =>
    `/${API_VERSION}/fleet/vpp_tokens/${id}/teams`,

  // MDM profile endpoints
  MDM_PROFILES: `/${API_VERSION}/fleet/mdm/profiles`,
  MDM_PROFILE: (id: string) => `/${API_VERSION}/fleet/mdm/profiles/${id}`,

  MDM_UPDATE_APPLE_SETTINGS: `/${API_VERSION}/fleet/mdm/apple/settings`,
  PROFILES_STATUS_SUMMARY: `/${API_VERSION}/fleet/configuration_profiles/summary`,
  DISK_ENCRYPTION: `/${API_VERSION}/fleet/disk_encryption`,
  MDM_APPLE_SSO: `/${API_VERSION}/fleet/mdm/sso`,
  MDM_APPLE_ENROLLMENT_PROFILE: (token: string, ref?: string) => {
    const query = new URLSearchParams({ token });
    if (ref) {
      query.append("enrollment_reference", ref);
    }
    return `/api/mdm/apple/enroll?${query}`;
  },
  MDM_APPLE_SETUP_ENROLLMENT_PROFILE: `/${API_VERSION}/fleet/mdm/apple/enrollment_profile`,
  MDM_BOOTSTRAP_PACKAGE_METADATA: (teamId: number) =>
    `/${API_VERSION}/fleet/mdm/bootstrap/${teamId}/metadata`,
  MDM_BOOTSTRAP_PACKAGE: `/${API_VERSION}/fleet/mdm/bootstrap`,
  MDM_BOOTSTRAP_PACKAGE_SUMMARY: `/${API_VERSION}/fleet/mdm/bootstrap/summary`,
  MDM_SETUP: `/${API_VERSION}/fleet/mdm/apple/setup`,
  MDM_EULA: (token: string) => `/${API_VERSION}/fleet/mdm/setup/eula/${token}`,
  MDM_EULA_UPLOAD: `/${API_VERSION}/fleet/mdm/setup/eula`,
  MDM_EULA_METADATA: `/${API_VERSION}/fleet/mdm/setup/eula/metadata`,
  HOST_MDM: (id: number) => `/${API_VERSION}/fleet/hosts/${id}/mdm`,
  HOST_MDM_UNENROLL: (id: number) =>
    `/${API_VERSION}/fleet/mdm/hosts/${id}/unenroll`,
  HOST_ENCRYPTION_KEY: (id: number) =>
    `/${API_VERSION}/fleet/hosts/${id}/encryption_key`,

  ME: `/${API_VERSION}/fleet/me`,

  // Disk encryption endpoints
  UPDATE_DISK_ENCRYPTION: `/${API_VERSION}/fleet/disk_encryption`,

  // Setup experiece endpoints
  MDM_SETUP_EXPERIENCE: `/${API_VERSION}/fleet/setup_experience`,
  MDM_SETUP_EXPERIENCE_SOFTWARE: `/${API_VERSION}/fleet/setup_experience/software`,
  MDM_SETUP_EXPERIENCE_SCRIPT: `/${API_VERSION}/fleet/setup_experience/script`,

  // OS Version endpoints
  OS_VERSIONS: `/${API_VERSION}/fleet/os_versions`,
  OS_VERSION: (id: number) => `/${API_VERSION}/fleet/os_versions/${id}`,

  OSQUERY_OPTIONS: `/${API_VERSION}/fleet/spec/osquery_options`,
  PACKS: `/${API_VERSION}/fleet/packs`,
  PERFORM_REQUIRED_PASSWORD_RESET: `/${API_VERSION}/fleet/perform_required_password_reset`,
  QUERIES: `/${API_VERSION}/fleet/queries`,
  QUERY_REPORT: (id: number) => `/${API_VERSION}/fleet/queries/${id}/report`,
  RESET_PASSWORD: `/${API_VERSION}/fleet/reset_password`,
  LIVE_QUERY: `/${API_VERSION}/fleet/queries/run`,
  SCHEDULE_QUERY: `/${API_VERSION}/fleet/packs/schedule`,
  SCHEDULED_QUERIES: (packId: number): string => {
    return `/${API_VERSION}/fleet/packs/${packId}/scheduled`;
  },
  SETUP: `/v1/setup`, // not a typo - hasn't been updated yet

  // Software endpoints
  SOFTWARE: `/${API_VERSION}/fleet/software`,
  SOFTWARE_TITLES: `/${API_VERSION}/fleet/software/titles`,
  SOFTWARE_TITLE: (id: number) => `/${API_VERSION}/fleet/software/titles/${id}`,
  EDIT_SOFTWARE_PACKAGE: (id: number) =>
    `/${API_VERSION}/fleet/software/titles/${id}/package`,
  SOFTWARE_VERSIONS: `/${API_VERSION}/fleet/software/versions`,
  SOFTWARE_VERSION: (id: number) =>
    `/${API_VERSION}/fleet/software/versions/${id}`,
  SOFTWARE_PACKAGE_ADD: `/${API_VERSION}/fleet/software/package`,
  SOFTWARE_PACKAGE_TOKEN: (id: number) =>
    `/${API_VERSION}/fleet/software/titles/${id}/package/token`,
  SOFTWARE_INSTALL_RESULTS: (uuid: string) =>
    `/${API_VERSION}/fleet/software/install/${uuid}/results`,
  SOFTWARE_PACKAGE_INSTALL: (id: number) =>
    `/${API_VERSION}/fleet/software/packages/${id}`,
  SOFTWARE_AVAILABLE_FOR_INSTALL: (id: number) =>
    `/${API_VERSION}/fleet/software/titles/${id}/available_for_install`,
  SOFTWARE_FLEET_MAINTAINED_APPS: `/${API_VERSION}/fleet/software/fleet_maintained_apps`,
  SOFTWARE_FLEET_MAINTAINED_APP: (id: number) =>
    `/${API_VERSION}/fleet/software/fleet_maintained_apps/${id}`,

  // AI endpoints
  AUTOFILL_POLICY: `/${API_VERSION}/fleet/autofill/policy`,

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
  TEAM_USERS: (teamId: number): string => {
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

  // Vulnerabilities endpoints
  VULNERABILITIES: `/${API_VERSION}/fleet/vulnerabilities`,
  VULNERABILITY: (cve: string) =>
    `/${API_VERSION}/fleet/vulnerabilities/${cve}`,

  // Script endpoints
  HOST_SCRIPTS: (id: number) => `/${API_VERSION}/fleet/hosts/${id}/scripts`,
  SCRIPTS: `/${API_VERSION}/fleet/scripts`,
  SCRIPT: (id: number) => `/${API_VERSION}/fleet/scripts/${id}`,
  SCRIPT_RESULT: (executionId: string) =>
    `/${API_VERSION}/fleet/scripts/results/${executionId}`,
  SCRIPT_RUN: `/${API_VERSION}/fleet/scripts/run`,

  COMMANDS_RESULTS: `/${API_VERSION}/fleet/commands/results`,
};
