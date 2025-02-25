import PropTypes from "prop-types";
import {
  IAppleDeviceUpdates,
  IConfigFeatures,
  IWebhookSettings,
} from "./config";
import enrollSecretInterface, { IEnrollSecret } from "./enroll_secret";
import { ITeamIntegrations } from "./integration";
import { UserRole } from "./user";

export default PropTypes.shape({
  id: PropTypes.number.isRequired,
  created_at: PropTypes.string,
  name: PropTypes.string.isRequired,
  description: PropTypes.string,
  agent_options: PropTypes.object, // eslint-disable-line react/forbid-prop-types
  role: PropTypes.any, // eslint-disable-line react/forbid-prop-types
  // role value is included when the team is in the context of a user
  user_count: PropTypes.number,
  host_count: PropTypes.number,
  secrets: PropTypes.arrayOf(enrollSecretInterface),
});

/**
 * The id, name, description, and host count for a team entity
 */
export interface ITeamSummary {
  id: number;
  name: string;
  description?: string;
  host_count?: number;
}

/**
 * The shape of a team entity excluding integrations and webhook settings
 */
export interface ITeam extends ITeamSummary {
  uuid?: string;
  display_text?: string;
  count?: number;
  created_at?: string;
  features?: IConfigFeatures;
  agent_options?: {
    [key: string]: any;
  };
  user_count?: number;
  host_count?: number;
  secrets?: IEnrollSecret[];
  role?: UserRole; // role value is included when the team is in the context of a user
  mdm?: {
    enable_disk_encryption: boolean;
    macos_updates: IAppleDeviceUpdates;
    ios_updates: IAppleDeviceUpdates;
    ipados_updates: IAppleDeviceUpdates;
    macos_settings: {
      custom_settings: null; // TODO: types?
      enable_disk_encryption: boolean;
    };
    macos_setup: {
      bootstrap_package: string | null;
      enable_end_user_authentication: boolean;
      macos_setup_assistant: string | null;
      enable_release_device_manually: boolean | null;
    };
    windows_updates: {
      deadline_days: number | null;
      grace_period_days: number | null;
    };
  };
  host_expiry_settings?: {
    host_expiry_enabled: boolean;
    host_expiry_window: number; // days
  };
}

/**
 * The webhook settings of a team
 */
export type ITeamWebhookSettings = Pick<
  IWebhookSettings,
  "vulnerabilities_webhook" | "failing_policies_webhook" | "host_status_webhook"
>;

/**
 * The integrations and webhook settings of a team
 */
export interface ITeamAutomationsConfig {
  webhook_settings: ITeamWebhookSettings;
  integrations: ITeamIntegrations;
}

/**
 * The shape of a team entity including integrations and webhook settings
 */
export type ITeamConfig = ITeam & ITeamAutomationsConfig;

/**
 * The shape of a new user to add to a team
 */
export interface INewTeamUser {
  id: number;
  role: UserRole;
}

/**
 * The shape of the body expected from the API when adding new users to teams
 */
export interface INewTeamUsersBody {
  users: INewTeamUser[];
}
export interface IRemoveTeamUserBody {
  users: { id?: number }[];
}
interface INewTeamSecret {
  team_id: number;
  secret: string;
  created_at?: string;
}
export interface INewTeamSecretBody {
  secrets: INewTeamSecret[];
}
export interface IRemoveTeamSecretBody {
  secrets: { secret: string }[];
}

export const API_ALL_TEAMS_ID = undefined;
export const APP_CONTEXT_ALL_TEAMS_ID = -1;
export const APP_CONTEXT_ALL_TEAMS_SUMMARY: ITeamSummary = {
  id: APP_CONTEXT_ALL_TEAMS_ID,
  name: "All teams",
} as const;

export const API_NO_TEAM_ID = 0;
export const APP_CONTEXT_NO_TEAM_ID = 0;
export const APP_CONTEXT_NO_TEAM_SUMMARY: ITeamSummary = {
  id: APP_CONTEXT_NO_TEAM_ID,
  name: "No team",
} as const;

export const isAnyTeamSelected = (currentTeamId?: number) =>
  currentTeamId !== undefined && currentTeamId > APP_CONTEXT_NO_TEAM_ID;
