import { flatMap, omit, pick, size } from "lodash";
import md5 from "js-md5";
import moment from "moment";

const ORG_INFO_ATTRS = ["org_name", "org_logo_url"];
const ADMIN_ATTRS = [
  "email",
  "name",
  "password",
  "password_confirmation",
  "username",
];

export const addGravatarUrlToResource = (resource: any): any => {
  const { email } = resource;

  const emailHash = md5(email.toLowerCase());
  const gravatarURL = `https://www.gravatar.com/avatar/${emailHash}?d=blank&size=200`;

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
    description:
      "Hosts that have recently checked-in to Fleet and are ready to run queries.",
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
    "kolide_server_url",
    "osquery_enroll_secret",
    "live_query_disabled",
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
  };
};

const formatLabelResponse = (response: any): { [index: string]: any } => {
  const labelTypeForDisplayText: { [index: string]: any } = {
    "All Hosts": "all",
    "MS Windows": "platform",
    "CentOS Linux": "platform",
    macOS: "platform",
    "Ubuntu Linux": "platform",
    "Red Hat Linux": "platform",
  };

  const labels = response.labels.map((label: any) => {
    return {
      ...label,
      slug: labelSlug(label),
      type: labelTypeForDisplayText[label.display_text] || "custom",
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

  if (appendID) {
    return { host_ids: hosts, label_ids: labels };
  }

  return { hosts, labels };
};

export const formatScheduledQueryForServer = (scheduledQuery: any) => {
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

const setupData = (formData: any) => {
  const orgInfo = pick(formData, ORG_INFO_ATTRS);
  const adminInfo = pick(formData, ADMIN_ATTRS);

  return {
    kolide_server_url: formData.kolide_server_url,
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

const inMilliseconds = (nanoseconds: number): number => {
  return nanoseconds / NANOSECONDS_PER_MILLISECOND;
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
  // July 28, 2016 is the date of the initial commit to kolide/fleet.
  if (detailUpdated < "2016-07-28T00:00:00Z") {
    return "Never";
  }

  return moment(detailUpdated).fromNow();
};

export default {
  addGravatarUrlToResource,
  formatConfigDataForServer,
  formatLabelResponse,
  formatScheduledQueryForClient,
  formatScheduledQueryForServer,
  formatSelectedTargetsForApi,
  humanHostUptime,
  humanHostLastSeen,
  humanHostEnrolled,
  humanHostMemory,
  humanHostDetailUpdated,
  labelSlug,
  setupData,
};
