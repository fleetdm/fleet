import { IConfig } from "interfaces/config";

export const DEFAULT_TRANSPARENCY_URL = "https://fleetdm.com/transparency";

export interface IAppConfigFormProps {
  appConfig: IConfig;
  isPremiumTier?: boolean;
  isUpdatingSettings?: boolean;
  handleSubmit: any;
}

export interface IFormField {
  name: string;
  value: string | boolean | number;
}

export interface IAppConfigFormErrors {
  metadata_url?: string | null;
  entity_id?: string | null;
  idp_name?: string | null;
  server_url?: string | null;
  org_name?: string | null;
  org_logo_url?: string | null;
  idp_image_url?: string | null;
  sender_address?: string | null;
  server?: string | null;
  server_port?: string | null;
  user_name?: string | null;
  password?: string | null;
  destination_url?: string | null;
  host_expiry_window?: string | null;
  agent_options?: string | null;
  transparency_url?: string | null;
}

export const authMethodOptions = [
  { label: "Plain", value: "authmethod_plain" },
  { label: "Cram MD5", value: "authmethod_cram_md5" },
  { label: "Login", value: "authmethod_login" },
];

export const authTypeOptions = [
  { label: "Username and Password", value: "authtype_username_password" },
  { label: "None", value: "authtype_none" },
];

export const percentageOfHosts = [
  { label: "1%", value: 1 },
  { label: "5%", value: 5 },
  { label: "10%", value: 10 },
  { label: "25%", value: 25 },
];

export const numberOfDays = [
  { label: "1 day", value: 1 },
  { label: "3 days", value: 3 },
  { label: "7 days", value: 7 },
  { label: "14 days", value: 14 },
];

export const hostStatusPreview = {
  text:
    "More than X% of your hosts have not checked into Fleet for more than Y days. You’ve been sent this message because the Host status webhook is enabled in your Fleet instance.",
  data: {
    unseen_hosts: 1,
    total_hosts: 2,
    days_unseen: 3,
  },
};
export const usageStatsPreview = {
  anonymousIdentifier: "9pnzNmrES3mQG66UQtd29cYTiX2+fZ4CYxDvh495720=",
  fleetVersion: "x.x.x",
  licenseTier: "free",
  organization: "Fleet",
  numHostsEnrolled: 999,
  numUsers: 999,
  numTeams: 999,
  numPolicies: 999,
  numLabels: 999,
  softwareInventoryEnabled: true,
  vulnDetectionEnabled: true,
  systemUsersEnabled: true,
  hostStatusWebhookEnabled: true,
  numWeeklyActiveUsers: 999,
  hostsEnrolledByOperatingSystem: {
    macos: [
      {
        version: "12.3.1",
        numEnrolled: 999,
      },
    ],
    windows: [
      {
        version: "10, version 21H2 (W)",
        numEnrolled: 999,
      },
    ],
    ubuntuLinux: [
      {
        version: "22.04 'Jammy Jellyfish' (LTS)",
        numEnrolled: 999,
      },
    ],
    centosLinux: [
      {
        version: "12.3.1",
        numEnrolled: 999,
      },
    ],
    debianLinux: [
      {
        version: "11 (Bullseye)",
        numEnrolled: 999,
      },
    ],
    redhatLinux: [
      {
        version: "9",
        numEnrolled: 999,
      },
    ],
    amazonLinux: [
      {
        version: "AMI",
        numEnrolled: 999,
      },
    ],
  },
  storedErrors: [
    {
      count: 3,
      loc: [
        "github.com/fleetdm/fleet/v4/server/example.example:12",
        "github.com/fleetdm/fleet/v4/server/example.example:130",
      ],
    },
  ],
  numHostsNotResponding: 9,
};

export default {
  authMethodOptions,
  authTypeOptions,
  percentageOfHosts,
  numberOfDays,
  hostStatusPreview,
  usageStatsPreview,
};
