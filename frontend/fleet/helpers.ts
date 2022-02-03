import { flatMap, omit, pick, size, memoize } from "lodash";
import md5 from "js-md5";
import moment from "moment";
import yaml from "js-yaml";

import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { IUser } from "interfaces/user";
import { IPackQueryFormData } from "interfaces/scheduled_query";

import stringUtils from "utilities/strings";
import sortUtils from "utilities/sort";
import {
  DEFAULT_GRAVATAR_LINK,
  PLATFORM_LABEL_DISPLAY_TYPES,
} from "utilities/constants";
import { IScheduledQueryStats } from "interfaces/scheduled_query_stats";

const ORG_INFO_ATTRS = ["org_name", "org_logo_url"];
const ADMIN_ATTRS = ["email", "name", "password", "password_confirmation"];

export const addGravatarUrlToResource = (resource: any): any => {
  const { email } = resource;

  const emailHash = md5(email.toLowerCase());
  const gravatarURL = `https://www.gravatar.com/avatar/${emailHash}?d=${encodeURIComponent(
    DEFAULT_GRAVATAR_LINK
  )}&size=200`;
  return {
    ...resource,
    gravatarURL,
  };
};

const labelSlug = (label: any): string => {
  const { id, name } = label;

  if (name === "All Hosts") {
    return "all-hosts";
  }

  return `labels/${id}`;
};

const labelStubs = [
  {
    id: "new",
    count: 0,
    description: "Hosts that have been enrolled to Fleet in the last 24 hours.",
    display_text: "New",
    slug: "new",
    statusLabelKey: "new_count",
    title_description: "(added in last 24hrs)",
    type: "status",
  },
  {
    id: "online",
    count: 0,
    description: "Hosts that have recently checked-in to Fleet.",
    display_text: "Online",
    slug: "online",
    statusLabelKey: "online_count",
    type: "status",
  },
  {
    id: "offline",
    count: 0,
    description: "Hosts that have not checked-in to Fleet recently.",
    display_text: "Offline",
    slug: "offline",
    statusLabelKey: "offline_count",
    type: "status",
  },
  {
    id: "mia",
    count: 0,
    description: "Hosts that have not been seen by Fleet in more than 30 days.",
    display_text: "MIA",
    slug: "mia",
    statusLabelKey: "mia_count",
    title_description: "(offline > 30 days)",
    type: "status",
  },
];

const filterTarget = (targetType: string) => {
  return (target: any) => {
    return target.target_type === targetType ? [target.id] : [];
  };
};

export const formatConfigDataForServer = (config: any): any => {
  const orgInfoAttrs = pick(config, ["org_logo_url", "org_name"]);
  const serverSettingsAttrs = pick(config, [
    "server_url",
    "osquery_enroll_secret",
    "live_query_disabled",
    "enable_analytics",
  ]);
  const smtpSettingsAttrs = pick(config, [
    "authentication_method",
    "authentication_type",
    "domain",
    "enable_ssl_tls",
    "enable_start_tls",
    "password",
    "port",
    "sender_address",
    "server",
    "user_name",
    "verify_ssl_certs",
    "enable_smtp",
  ]);
  const ssoSettingsAttrs = pick(config, [
    "entity_id",
    "issuer_uri",
    "idp_image_url",
    "metadata",
    "metadata_url",
    "idp_name",
    "enable_sso",
    "enable_sso_idp_login",
  ]);
  const hostExpirySettingsAttrs = pick(config, [
    "host_expiry_enabled",
    "host_expiry_window",
  ]);
  const webhookSettingsAttrs = pick(config, [
    "enable_host_status_webhook",
    "destination_url",
    "host_percentage",
    "days_count",
  ]);
  // because agent_options is already an object
  const agentOptionsSettingsAttrs = config.agent_options;

  const orgInfo = size(orgInfoAttrs) && { org_info: orgInfoAttrs };
  const serverSettings = size(serverSettingsAttrs) && {
    server_settings: serverSettingsAttrs,
  };
  const smtpSettings = size(smtpSettingsAttrs) && {
    smtp_settings: smtpSettingsAttrs,
  };
  const ssoSettings = size(ssoSettingsAttrs) && {
    sso_settings: ssoSettingsAttrs,
  };
  const hostExpirySettings = size(hostExpirySettingsAttrs) && {
    host_expiry_settings: hostExpirySettingsAttrs,
  };
  const agentOptionsSettings = size(agentOptionsSettingsAttrs) && {
    agent_options: yaml.load(agentOptionsSettingsAttrs),
  };
  const webhookSettings = size(webhookSettingsAttrs) && {
    webhook_settings: { host_status_webhook: webhookSettingsAttrs }, // nested to server
  };

  if (hostExpirySettings) {
    hostExpirySettings.host_expiry_settings.host_expiry_window = Number(
      hostExpirySettings.host_expiry_settings.host_expiry_window
    );
  }

  return {
    ...orgInfo,
    ...serverSettings,
    ...smtpSettings,
    ...ssoSettings,
    ...hostExpirySettings,
    ...agentOptionsSettings,
    ...webhookSettings,
  };
};

// TODO: Finalize interface for config - see frontend\interfaces\config.ts
export const frontendFormattedConfig = (config: any) => {
  const {
    org_info: orgInfo,
    server_settings: serverSettings,
    smtp_settings: smtpSettings,
    sso_settings: ssoSettings,
    host_expiry_settings: hostExpirySettings,
    webhook_settings: { host_status_webhook: webhookSettings }, // unnested to frontend
    update_interval: updateInterval,
    license,
  } = config;

  if (config.agent_options) {
    config.agent_options = yaml.dump(config.agent_options);
  }

  return {
    ...orgInfo,
    ...serverSettings,
    ...smtpSettings,
    ...ssoSettings,
    ...hostExpirySettings,
    ...webhookSettings,
    ...updateInterval,
    ...license,
    agent_options: config.agent_options,
  };
};

const formatLabelResponse = (response: any): ILabel[] => {
  const labels = response.labels.map((label: any) => {
    return {
      ...label,
      slug: labelSlug(label),
      type: PLATFORM_LABEL_DISPLAY_TYPES[label.display_text] || "custom",
      target_type: "labels",
    };
  });

  return labels.concat(labelStubs);
};

export const formatSelectedTargetsForApi = (
  selectedTargets: any,
  appendID = false
) => {
  const targets = selectedTargets || [];
  const hosts = flatMap(targets, filterTarget("hosts"));
  const labels = flatMap(targets, filterTarget("labels"));
  const teams = flatMap(targets, filterTarget("teams"));

  if (appendID) {
    return { host_ids: hosts, label_ids: labels, team_ids: teams };
  }

  return { hosts, labels, teams };
};

export const formatScheduledQueryForServer = (
  scheduledQuery: IPackQueryFormData
) => {
  const {
    interval,
    logging_type: loggingType,
    pack_id: packID,
    platform,
    query_id: queryID,
    shard,
  } = scheduledQuery;
  const result = omit(scheduledQuery, ["logging_type"]);

  if (platform === "all") {
    result.platform = "";
  }

  if (interval) {
    result.interval = Number(interval);
  }

  if (loggingType) {
    result.removed = loggingType === "differential";
    result.snapshot = loggingType === "snapshot";
  }

  if (packID) {
    result.pack_id = Number(packID);
  }

  if (queryID) {
    result.query_id = Number(queryID);
  }

  if (shard) {
    result.shard = Number(shard);
  }

  return result;
};

export const formatScheduledQueryForClient = (scheduledQuery: any): any => {
  if (scheduledQuery.platform === "") {
    scheduledQuery.platform = "all";
  }

  if (scheduledQuery.snapshot) {
    scheduledQuery.logging_type = "snapshot";
  } else {
    scheduledQuery.snapshot = false;
    if (scheduledQuery.removed === false) {
      scheduledQuery.logging_type = "differential_ignore_removals";
    } else {
      // If both are unset, we should default to differential (like osquery does)
      scheduledQuery.logging_type = "differential";
    }
  }

  if (scheduledQuery.shard === null) {
    scheduledQuery.shard = undefined;
  }

  return scheduledQuery;
};

export const formatGlobalScheduledQueryForServer = (scheduledQuery: any) => {
  const {
    interval,
    logging_type: loggingType,
    platform,
    query_id: queryID,
    shard,
  } = scheduledQuery;
  const result = omit(scheduledQuery, ["logging_type"]);

  if (platform === "all") {
    result.platform = "";
  }

  if (interval) {
    result.interval = Number(interval);
  }

  if (loggingType) {
    result.removed = loggingType === "differential";
    result.snapshot = loggingType === "snapshot";
  }

  if (queryID) {
    result.query_id = Number(queryID);
  }

  if (shard) {
    result.shard = Number(shard);
  }

  return result;
};

export const formatGlobalScheduledQueryForClient = (
  scheduledQuery: any
): any => {
  if (scheduledQuery.platform === "") {
    scheduledQuery.platform = "all";
  }

  if (scheduledQuery.snapshot) {
    scheduledQuery.logging_type = "snapshot";
  } else {
    scheduledQuery.snapshot = false;
    if (scheduledQuery.removed === false) {
      scheduledQuery.logging_type = "differential_ignore_removals";
    } else {
      // If both are unset, we should default to differential (like osquery does)
      scheduledQuery.logging_type = "differential";
    }
  }

  if (scheduledQuery.shard === null) {
    scheduledQuery.shard = undefined;
  }

  return scheduledQuery;
};

export const formatTeamScheduledQueryForServer = (scheduledQuery: any) => {
  const {
    interval,
    logging_type: loggingType,
    platform,
    query_id: queryID,
    shard,
    team_id: teamID,
  } = scheduledQuery;
  const result = omit(scheduledQuery, ["logging_type"]);

  if (platform === "all") {
    result.platform = "";
  }

  if (interval) {
    result.interval = Number(interval);
  }

  if (loggingType) {
    result.removed = loggingType === "differential";
    result.snapshot = loggingType === "snapshot";
  }

  if (queryID) {
    result.query_id = Number(queryID);
  }

  if (shard) {
    result.shard = Number(shard);
  }

  if (teamID) {
    result.query_id = Number(teamID);
  }

  return result;
};

export const formatTeamScheduledQueryForClient = (scheduledQuery: any): any => {
  if (scheduledQuery.platform === "") {
    scheduledQuery.platform = "all";
  }

  if (scheduledQuery.snapshot) {
    scheduledQuery.logging_type = "snapshot";
  } else {
    scheduledQuery.snapshot = false;
    if (scheduledQuery.removed === false) {
      scheduledQuery.logging_type = "differential_ignore_removals";
    } else {
      // If both are unset, we should default to differential (like osquery does)
      scheduledQuery.logging_type = "differential";
    }
  }

  if (scheduledQuery.shard === null) {
    scheduledQuery.shard = undefined;
  }

  return scheduledQuery;
};

export const formatTeamForClient = (team: any): any => {
  if (team.display_text === undefined) {
    team.display_text = team.name;
  }
  return team;
};

export const formatPackForClient = (pack: any): any => {
  pack.host_ids ||= [];
  pack.label_ids ||= [];
  pack.team_ids ||= [];
  return pack;
};

export const generateRole = (
  teams: ITeam[],
  globalRole: string | null
): string => {
  if (globalRole === null) {
    const listOfRoles: (string | undefined)[] = teams.map((team) => team.role);

    if (teams.length === 0) {
      // no global role and no teams
      return "Unassigned";
    } else if (teams.length === 1) {
      // no global role and only one team
      return stringUtils.capitalize(teams[0].role ?? "");
    } else if (
      listOfRoles.every(
        (role: string | undefined): boolean => role === "maintainer"
      )
    ) {
      // only team maintainers
      return "Maintainer";
    } else if (
      listOfRoles.every(
        (role: string | undefined): boolean => role === "observer"
      )
    ) {
      // only team observers
      return "Observer";
    }

    return "Various"; // no global role and multiple teams
  }

  if (teams.length === 0) {
    // global role and no teams
    return stringUtils.capitalize(globalRole);
  }
  return "Various"; // global role and one or more teams
};

export const generateTeam = (
  teams: ITeam[],
  globalRole: string | null
): string => {
  if (globalRole === null) {
    if (teams.length === 0) {
      // no global role and no teams
      return "No Team";
    } else if (teams.length === 1) {
      // no global role and only one team
      return teams[0].name;
    }
    return `${teams.length} teams`; // no global role and multiple teams
  }

  if (teams.length === 0) {
    // global role and no teams
    return "Global";
  }
  return `${teams.length + 1} teams`; // global role and one or more teams
};

export const greyCell = (roleOrTeamText: string): string => {
  const GREYED_TEXT = ["Global", "Unassigned", "Various", "No Team"];

  if (
    GREYED_TEXT.includes(roleOrTeamText) ||
    roleOrTeamText.includes(" teams")
  ) {
    return "grey-cell";
  }
  return "";
};

const setupData = (formData: any) => {
  const orgInfo = pick(formData, ORG_INFO_ATTRS);
  const adminInfo = pick(formData, ADMIN_ATTRS);

  return {
    server_url: formData.server_url,
    org_info: {
      ...orgInfo,
    },
    admin: {
      admin: true,
      ...adminInfo,
    },
  };
};

const BYTES_PER_GIGABYTE = 1074000000;
const NANOSECONDS_PER_MILLISECOND = 1000000;

const inGigaBytes = (bytes: number): string => {
  return (bytes / BYTES_PER_GIGABYTE).toFixed(1);
};

export const inMilliseconds = (nanoseconds: number): number => {
  return nanoseconds / NANOSECONDS_PER_MILLISECOND;
};

export const humanTimeAgo = (dateSince: string): number => {
  const now = moment();
  const mDateSince = moment(dateSince);

  return now.diff(mDateSince, "days");
};

export const humanHostUptime = (uptimeInNanoseconds: number): string => {
  const milliseconds = inMilliseconds(uptimeInNanoseconds);

  return moment.duration(milliseconds, "milliseconds").humanize();
};

export const humanHostLastSeen = (lastSeen: string): string => {
  return moment(lastSeen).format("MMM D YYYY, HH:mm:ss");
};

export const humanHostEnrolled = (enrolled: string): string => {
  return moment(enrolled).format("MMM D YYYY, HH:mm:ss");
};

export const humanHostMemory = (bytes: number): string => {
  return `${inGigaBytes(bytes)} GB`;
};

export const humanHostDetailUpdated = (detailUpdated: string): string => {
  // Handles the case when a host has checked in to Fleet but
  // its details haven't been updated.
  // July 28, 2016 is the date of the initial commit to fleet/fleet.
  if (detailUpdated < "2016-07-28T00:00:00Z") {
    return "Never";
  }

  return moment(detailUpdated).fromNow();
};

export const hostTeamName = (teamName: string | null): string => {
  if (!teamName) {
    return "No team";
  }

  return teamName;
};

export const humanQueryLastRun = (lastRun: string): string => {
  // Handles the case when a query has never been ran.
  // July 28, 2016 is the date of the initial commit to fleet/fleet.
  if (lastRun < "2016-07-28T00:00:00Z") {
    return "Has not run";
  }

  return moment(lastRun).fromNow();
};

export const licenseExpirationWarning = (expiration: string): boolean => {
  return moment(moment()).isAfter(expiration);
};

// IQueryStats became any when adding in IGlobalScheduledQuery and ITeamScheduledQuery
export const performanceIndicator = (
  scheduledQueryStats: IScheduledQueryStats
): string => {
  if (
    !scheduledQueryStats.total_executions ||
    scheduledQueryStats.total_executions === 0 ||
    scheduledQueryStats.total_executions === null
  ) {
    return "Undetermined";
  }

  if (
    typeof scheduledQueryStats.user_time_p50 === "number" &&
    typeof scheduledQueryStats.system_time_p50 === "number"
  ) {
    const indicator =
      scheduledQueryStats.user_time_p50 + scheduledQueryStats.system_time_p50;

    if (indicator < 2000) {
      return "Minimal";
    }
    if (indicator < 4000) {
      return "Considerable";
    }
  }
  return "Excessive";
};

export const secondsToHms = (d: number): string => {
  const h = Math.floor(d / 3600);
  const m = Math.floor((d % 3600) / 60);
  const s = Math.floor((d % 3600) % 60);

  const hDisplay = h > 0 ? h + (h === 1 ? " hr " : " hrs ") : "";
  const mDisplay = m > 0 ? m + (m === 1 ? " min " : " mins ") : "";
  const sDisplay = s > 0 ? s + (s === 1 ? " sec" : " secs") : "";
  return hDisplay + mDisplay + sDisplay;
};

export const secondsToDhms = (d: number): string => {
  if (d === 604800) {
    return "1 week";
  }
  const day = Math.floor(d / (3600 * 24));
  const h = Math.floor((d % (3600 * 24)) / 3600);
  const m = Math.floor((d % 3600) / 60);
  const s = Math.floor(d % 60);

  const dDisplay = day > 0 ? day + (day === 1 ? " day " : " days ") : "";
  const hDisplay = h > 0 ? h + (h === 1 ? " hour " : " hours ") : "";
  const mDisplay = m > 0 ? m + (m === 1 ? " minute " : " minutes ") : "";
  const sDisplay = s > 0 ? s + (s === 1 ? " second" : " seconds") : "";
  return dDisplay + hDisplay + mDisplay + sDisplay;
};

// TODO: Type any because ts files missing the following properties from type 'JSON': parse, stringify, [Symbol.toStringTag]
export const syntaxHighlight = (json: any): string => {
  let jsonStr: string = JSON.stringify(json, undefined, 2);
  jsonStr = jsonStr
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
  /* eslint-disable no-useless-escape */
  return jsonStr.replace(
    /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
    function (match) {
      let cls = "number";
      if (/^"/.test(match)) {
        if (/:$/.test(match)) {
          cls = "key";
        } else {
          cls = "string";
        }
      } else if (/true|false/.test(match)) {
        cls = "boolean";
      } else if (/null/.test(match)) {
        cls = "null";
      }
      return `<span class="${cls}">${match}</span>`;
    }
  );
  /* eslint-enable no-useless-escape */
};

export const getSortedTeamOptions = memoize((teams: ITeam[]) =>
  teams
    .map((team) => {
      return {
        disabled: false,
        label: team.name,
        value: team.id,
      };
    })
    .sort((a, b) => sortUtils.caseInsensitiveAsc(a.label, b.label))
);

export const getValidatedTeamId = (
  teams: ITeam[],
  teamId: number,
  currentUser: IUser | null,
  isOnGlobalTeam: boolean
): number => {
  let currentUserTeams: ITeam[] = [];
  if (isOnGlobalTeam) {
    currentUserTeams = teams;
  } else if (currentUser && currentUser.teams) {
    currentUserTeams = currentUser.teams;
  }

  const currentUserTeamIds = currentUserTeams.map((t) => t.id);
  const validatedTeamId =
    !isNaN(teamId) && teamId > 0 && currentUserTeamIds.includes(teamId)
      ? teamId
      : 0;

  return validatedTeamId;
};

export default {
  addGravatarUrlToResource,
  formatConfigDataForServer,
  formatLabelResponse,
  formatScheduledQueryForClient,
  formatScheduledQueryForServer,
  formatGlobalScheduledQueryForClient,
  formatGlobalScheduledQueryForServer,
  formatTeamScheduledQueryForClient,
  formatTeamScheduledQueryForServer,
  formatSelectedTargetsForApi,
  generateRole,
  generateTeam,
  greyCell,
  humanHostUptime,
  humanHostLastSeen,
  humanHostEnrolled,
  humanHostMemory,
  humanHostDetailUpdated,
  hostTeamName,
  humanQueryLastRun,
  inMilliseconds,
  licenseExpirationWarning,
  secondsToHms,
  secondsToDhms,
  labelSlug,
  setupData,
  frontendFormattedConfig,
  syntaxHighlight,
  getValidatedTeamId,
};
