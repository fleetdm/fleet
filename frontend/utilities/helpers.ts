import {
  isEmpty,
  flatMap,
  omit,
  pick,
  size,
  memoize,
  reduce,
  trim,
  trimEnd,
  union,
} from "lodash";
import { buildQueryStringFromParams } from "utilities/url";

import md5 from "js-md5";
import {
  formatDistanceToNow,
  formatDuration,
  intlFormat,
  intervalToDuration,
  isAfter,
} from "date-fns";
import yaml from "js-yaml";

import { IHost } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { IPack } from "interfaces/pack";
import {
  IScheduledQuery,
  IPackQueryFormData,
} from "interfaces/scheduled_query";
import {
  ISelectTargetsEntity,
  ISelectedTargets,
  IPackTargets,
} from "interfaces/target";
import { ITeam, ITeamSummary } from "interfaces/team";
import { IUser, UserRole } from "interfaces/user";

import stringUtils from "utilities/strings";
import sortUtils from "utilities/sort";
import {
  DEFAULT_EMPTY_CELL_VALUE,
  DEFAULT_GRAVATAR_LINK,
  DEFAULT_GRAVATAR_LINK_FALLBACK,
  DEFAULT_GRAVATAR_LINK_DARK,
  DEFAULT_GRAVATAR_LINK_DARK_FALLBACK,
  INITIAL_FLEET_DATE,
  PLATFORM_LABEL_DISPLAY_TYPES,
} from "utilities/constants";
import { IScheduledQueryStats } from "interfaces/scheduled_query_stats";

const ORG_INFO_ATTRS = ["org_name", "org_logo_url"];
const ADMIN_ATTRS = ["email", "name", "password", "password_confirmation"];

export const addGravatarUrlToResource = (resource: any): any => {
  const { email } = resource;
  const gravatarAvailable =
    localStorage.getItem("gravatar_available") !== "false"; // Only fallback if explicitly set to "false"

  const emailHash = md5(email.toLowerCase());

  let gravatar_url;
  let gravatar_url_dark;

  if (gravatarAvailable) {
    gravatar_url = `https://www.gravatar.com/avatar/${emailHash}?d=${encodeURIComponent(
      DEFAULT_GRAVATAR_LINK
    )}&size=200`;
    gravatar_url_dark = `https://www.gravatar.com/avatar/${emailHash}?d=${encodeURIComponent(
      DEFAULT_GRAVATAR_LINK_DARK
    )}&size=200`;
  } else {
    gravatar_url = DEFAULT_GRAVATAR_LINK_FALLBACK;
    gravatar_url_dark = DEFAULT_GRAVATAR_LINK_DARK_FALLBACK;
  }

  return {
    ...resource,
    gravatar_url,
    gravatar_url_dark,
  };
};

const labelSlug = (label: ILabel): string => {
  const { id, name } = label;

  if (name === "All Hosts") {
    return "all-hosts";
  }

  return `labels/${id}`;
};

const statusKey = [
  {
    id: "new",
    count: 0,
    description: "Hosts that have been enrolled to Fleet in the last 24 hours.",
    display_text: "New",
    title_description: "(added in last 24hrs)",
    type: "status",
  },
  {
    id: "online",
    count: 0,
    description: "Hosts that have recently checked-in to Fleet.",
    display_text: "Online",
    type: "status",
  },
  {
    id: "missing",
    count: 0,
    description: "Hosts that have not been online in 30 days or more.",
    display_text: "Missing",
    slug: "missing",
    statusLabelKey: "missing_count",
    type: "status",
  },
  {
    id: "offline",
    count: 0,
    description: "Hosts that have not checked-in to Fleet recently.",
    display_text: "Offline",
    type: "status",
  },
];

const isLabel = (target: ISelectTargetsEntity) => {
  return "label_type" in target;
};
const isHost = (target: ISelectTargetsEntity) => {
  return "display_name" in target;
};

const filterTarget = (targetType: string) => {
  return (target: ISelectTargetsEntity) => {
    const id =
      typeof target.id !== "number" ? parseInt(target.id, 10) : target.id;
    if ("target_type" in target) {
      return target.target_type === targetType && !isNaN(id) ? [id] : [];
    }
    switch (targetType) {
      case "hosts":
        return isHost(target) && !isNaN(id) ? [id] : [];
      case "labels":
        return isLabel(target) && !isNaN(id) ? [id] : [];
      case "teams":
        return !isHost(target) && !isLabel(target) && !isNaN(id) ? [id] : [];
      default:
        return [];
    }
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

export const formatFloatAsPercentage = (float: number): string => {
  const formatter = Intl.NumberFormat("en-US", {
    maximumSignificantDigits: 2,
    style: "percent",
  });

  return formatter.format(float);
};

const formatLabelResponse = (response: any): ILabel[] => {
  const labels = response.labels.map((label: ILabel) => {
    return {
      ...label,
      slug: labelSlug(label),
      type: PLATFORM_LABEL_DISPLAY_TYPES[label.display_text] || "custom",
      target_type: "labels",
    };
  });

  return labels;
};

export const formatSelectedTargetsForApi = (
  selectedTargets: ISelectTargetsEntity[]
): ISelectedTargets => {
  const targets = selectedTargets || [];
  // TODO: can flatMap be removed?
  const hostIds = flatMap(targets, filterTarget("hosts"));
  const labelIds = flatMap(targets, filterTarget("labels"));
  const teamIds = flatMap(targets, filterTarget("teams"));

  return {
    hosts: hostIds.sort(),
    labels: labelIds.sort(),
    teams: teamIds.sort(),
  };
};

export const formatPackTargetsForApi = (
  targets: ISelectTargetsEntity[]
): IPackTargets => {
  const { hosts, labels, teams } = formatSelectedTargetsForApi(targets);
  return { host_ids: hosts, label_ids: labels, team_ids: teams };
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

export const formatScheduledQueryForClient = (
  scheduledQuery: IScheduledQuery
): IScheduledQuery => {
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

export const formatGlobalScheduledQueryForServer = (
  scheduledQuery: IScheduledQuery
): IScheduledQuery => {
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
  scheduledQuery: IScheduledQuery
): IScheduledQuery => {
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

export const formatTeamScheduledQueryForServer = (
  scheduledQuery: IScheduledQuery
) => {
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

export const formatTeamScheduledQueryForClient = (
  scheduledQuery: IScheduledQuery
): IScheduledQuery => {
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

export const formatTeamForClient = (team: ITeam): ITeam => {
  if (team.display_text === undefined) {
    team.display_text = team.name;
  }
  return team;
};

export const formatPackForClient = (pack: IPack): IPack => {
  pack.host_ids ||= [];
  pack.label_ids ||= [];
  pack.team_ids ||= [];
  return pack;
};

export const generateRole = (
  teams: ITeam[],
  globalRole: UserRole | null
): UserRole => {
  if (globalRole === null) {
    const listOfRoles = teams.map<UserRole | undefined>((team) => team.role);

    if (teams.length === 0) {
      // no global role and no teams
      return "Unassigned";
    } else if (teams.length === 1) {
      // no global role and only one team
      return stringUtils.capitalizeRole(teams[0].role || "Unassigned");
    } else if (listOfRoles.every((role): boolean => role === "maintainer")) {
      // only team maintainers
      return "Maintainer";
    } else if (listOfRoles.every((role): boolean => role === "observer")) {
      // only team observers
      return "Observer";
    } else if (listOfRoles.every((role): boolean => role === "observer_plus")) {
      // only team observers plus
      return "Observer+";
    }

    return "Various"; // no global role and multiple teams
  }

  if (teams.length === 0) {
    // global role and no teams
    return stringUtils.capitalizeRole(globalRole);
  }
  return "Various"; // global role and one or more teams
};

export const generateTeam = (
  teams: ITeam[],
  globalRole: UserRole | null
): string => {
  if (globalRole === null) {
    if (teams.length === 0) {
      // no global role and no teams
      return "No team";
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

export const greyCell = (roleOrTeamText: string): boolean => {
  const GREYED_TEXT = ["Global", "Unassigned", "Various", "No team", "Unknown"];

  return (
    GREYED_TEXT.includes(roleOrTeamText) || roleOrTeamText.includes(" teams")
  );
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

export const humanHostLastRestart = (
  detailUpdatedAt: string,
  uptime: number | string
): string => {
  if (
    !detailUpdatedAt ||
    !uptime ||
    detailUpdatedAt === DEFAULT_EMPTY_CELL_VALUE ||
    detailUpdatedAt < INITIAL_FLEET_DATE ||
    typeof uptime !== "number"
  ) {
    return "Unavailable";
  }
  try {
    const currentDate = new Date();
    const updatedDate = new Date(detailUpdatedAt);
    const millisecondsLastUpdated =
      currentDate.getTime() - updatedDate.getTime();

    // Sum of calculated milliseconds since last updated with uptime
    const millisecondsLastRestart =
      millisecondsLastUpdated + uptime / NANOSECONDS_PER_MILLISECOND;

    const restartDate = new Date();
    restartDate.setMilliseconds(
      restartDate.getMilliseconds() - millisecondsLastRestart
    );

    return restartDate.toString();
  } catch {
    return "Unavailable";
  }
};

export const humanHostLastSeen = (lastSeen: string): string => {
  if (!lastSeen || lastSeen < INITIAL_FLEET_DATE) {
    return "Never";
  }
  if (lastSeen === "Unavailable") {
    return "Unavailable";
  }
  return formatDistanceToNow(new Date(lastSeen), { addSuffix: true });
};

export const humanHostEnrolled = (enrolled: string): string => {
  if (!enrolled || enrolled < INITIAL_FLEET_DATE) {
    return "Never";
  }
  return formatDistanceToNow(new Date(enrolled), { addSuffix: true });
};

export const humanHostMemory = (bytes: number): string => {
  return `${inGigaBytes(bytes)} GB`;
};

export const humanHostDetailUpdated = (detailUpdated?: string): string => {
  // Handles the case when a host has checked in to Fleet but
  // its details haven't been updated.
  if (!detailUpdated || detailUpdated < INITIAL_FLEET_DATE) {
    return "unavailable";
  }
  try {
    return formatDistanceToNow(new Date(detailUpdated), { addSuffix: true });
  } catch {
    return "unavailable";
  }
};

/** Unlike humanHost helper functions, there are no Fleet-related date restrictions */
export const humanLastSeen = (lastSeen: string): string => {
  if (!lastSeen) {
    return "Never";
  }
  if (lastSeen === "Unavailable") {
    return "Unavailable";
  }

  return formatDistanceToNow(new Date(lastSeen), { addSuffix: true });
};

export const internationalTimeFormat = (date: number | Date): string => {
  return intlFormat(
    date,
    {
      year: "numeric",
      month: "numeric",
      day: "numeric",
      hour: "numeric",
      minute: "numeric",
      second: "numeric",
    },
    { locale: window.navigator.languages[0] }
  );
};

const MAC_WINDOWS_DISK_ENCRYPTION_MESSAGES = {
  darwin: {
    enabled:
      "The disk is encrypted. The user must enter their<br/> password when they start their computer.",
    disabled:
      "The disk might be encrypted, but FileVault is off. The<br/> disk can be accessed without entering a password.",
  },
  windows: {
    enabled:
      "The disk is encrypted. If recently turned on,<br/> encryption could take awhile.",
    disabled: "The disk is unencrypted.",
  },
};

export const getHostDiskEncryptionTooltipMessage = (
  platform: "darwin" | "windows" | "chrome", // TODO: improve this type
  diskEncryptionEnabled = false
) => {
  if (platform === "chrome") {
    return "Fleet does not check for disk encryption on Chromebooks, as they are encrypted by default.";
  }

  if (!["windows", "darwin"].includes(platform)) {
    return "Disk encryption is enabled.";
  }
  return MAC_WINDOWS_DISK_ENCRYPTION_MESSAGES[platform][
    diskEncryptionEnabled ? "enabled" : "disabled"
  ];
};

export const hostTeamName = (teamName: string | null): string => {
  if (!teamName) {
    return "No team";
  }

  return teamName;
};

export const humanQueryLastRun = (lastRun: string): string => {
  // Handles the case when a query has never been ran.
  if (!lastRun || lastRun < INITIAL_FLEET_DATE) {
    return "Has not run";
  }

  try {
    return formatDistanceToNow(new Date(lastRun), { addSuffix: true });
  } catch {
    return "Unavailable";
  }
};

export const licenseExpirationWarning = (expiration: string): boolean => {
  return isAfter(new Date(), new Date(expiration));
};

export const readableDate = (date: string) => {
  const dateString = new Date(date);

  return new Intl.DateTimeFormat(navigator.language, {
    year: "numeric",
    month: "long",
    day: "numeric",
  }).format(dateString);
};

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

export const secondsToDhms = (s: number): string => {
  if (s === 604800) {
    return "1 week";
  }

  const duration = intervalToDuration({ start: 0, end: s * 1000 });
  return formatDuration(duration);
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

export const abbreviateTimeUnits = (str: string): string =>
  str.replace("minute", "min").replace("second", "sec");

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

// returns a mixture of props from host
export const normalizeEmptyValues = (
  hostData: Partial<IHost>
): Record<
  string,
  number | string | boolean | Record<string, number | string | boolean>
> => {
  return reduce(
    hostData,
    (result, value, key) => {
      if ((Number.isFinite(value) && value !== 0) || !isEmpty(value)) {
        Object.assign(result, { [key]: value });
      } else {
        Object.assign(result, { [key]: DEFAULT_EMPTY_CELL_VALUE });
      }
      return result;
    },
    {}
  );
};

export const wrapFleetHelper = (
  helperFn: (value: any) => string, // TODO: replace any with unknown and improve type narrowing by callers
  value: string
): string => {
  return value === DEFAULT_EMPTY_CELL_VALUE ? value : helperFn(value);
};

interface ILocationParams {
  pathPrefix?: string;
  routeTemplate?: string;
  routeParams?: { [key: string]: string };
  queryParams?: { [key: string]: string | number | undefined };
}

type RouteParams = Record<string, string>;

const createRouteString = (routeTemplate: string, routeParams: RouteParams) => {
  let routeString = "";
  if (!isEmpty(routeParams)) {
    routeString = reduce(
      routeParams,
      (string, value, key) => {
        return string.replace(`:${key}`, encodeURIComponent(value));
      },
      routeTemplate
    );
  }
  return routeString;
};

export const getNextLocationPath = ({
  pathPrefix = "",
  routeTemplate = "",
  routeParams = {},
  queryParams = {},
}: ILocationParams): string => {
  const routeString = createRouteString(routeTemplate, routeParams);
  const queryString = buildQueryStringFromParams(queryParams);

  const nextLocation = trimEnd(
    union(trim(pathPrefix, "/").split("/"), routeString.split("/")).join("/"),
    "/"
  );

  return queryString ? `/${nextLocation}?${queryString}` : `/${nextLocation}`;
};

export const getSoftwareBundleTooltipMarkup = (bundle: string) => {
  return `
        <span>
          <b>Bundle identifier: </b>
          <br />
          ${bundle}
        </span>
      `;
};

export default {
  addGravatarUrlToResource,
  formatConfigDataForServer,
  formatLabelResponse,
  formatFloatAsPercentage,
  formatScheduledQueryForClient,
  formatScheduledQueryForServer,
  formatGlobalScheduledQueryForClient,
  formatGlobalScheduledQueryForServer,
  formatTeamScheduledQueryForClient,
  formatTeamScheduledQueryForServer,
  formatSelectedTargetsForApi,
  formatPackTargetsForApi,
  generateRole,
  generateTeam,
  greyCell,
  humanHostLastSeen,
  humanHostEnrolled,
  humanHostMemory,
  humanHostDetailUpdated,
  humanLastSeen,
  internationalTimeFormat,
  getHostDiskEncryptionTooltipMessage,
  hostTeamName,
  humanQueryLastRun,
  inMilliseconds,
  licenseExpirationWarning,
  readableDate,
  secondsToHms,
  secondsToDhms,
  labelSlug,
  setupData,
  syntaxHighlight,
  normalizeEmptyValues,
  wrapFleetHelper,
};
