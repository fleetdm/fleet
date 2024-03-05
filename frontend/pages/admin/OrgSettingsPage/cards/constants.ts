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
  metadata?: string | null;
  metadata_url?: string | null;
  entity_id?: string | null;
  idp_name?: string | null;
  server_url?: string | null;
  org_name?: string | null;
  org_logo_url?: string | null;
  org_logo_url_light_background?: string | null;
  org_support_url?: string | null;
  idp_image_url?: string | null;
  sender_address?: string | null;
  server?: string | null;
  server_port?: string | null;
  user_name?: string | null;
  password?: string | null;
  destination_url?: string | null;
  days_count?: string | null;
  host_percentage?: string | null;
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

export default {
  authMethodOptions,
  authTypeOptions,
  percentageOfHosts,
  numberOfDays,
};
