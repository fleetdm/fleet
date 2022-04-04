import { IConfigNested } from "interfaces/config";

export interface IAppConfigFormProps {
  appConfig: IConfigNested;
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
  numHostsEnrolled: 12345,
  numUsers: 12,
  numTeams: 3,
  numPolicies: 5,
  numLabels: 20,
  softwareInventoryEnabled: true,
  vulnDetectionEnabled: true,
  systemUsersEnabled: true,
  hostStatusWebhookEnabled: true,
};

export default {
  authMethodOptions,
  authTypeOptions,
  percentageOfHosts,
  numberOfDays,
  hostStatusPreview,
  usageStatsPreview,
};
