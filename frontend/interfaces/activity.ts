import {
  IPastActivitiesResponse,
  IUpcomingActivitiesResponse,
} from "services/entities/activities";
import { IPolicy } from "./policy";
import { IQuery } from "./query";
import { IScheduledQueryStats } from "./scheduled_query_stats";
import { ITeamSummary } from "./team";
import { UserRole } from "./user";

export enum ActivityType {
  CreatedPack = "created_pack",
  DeletedPack = "deleted_pack",
  EditedPack = "edited_pack",
  CreatedPolicy = "created_policy",
  DeletedPolicy = "deleted_policy",
  EditedPolicy = "edited_policy",
  CreatedSavedQuery = "created_saved_query",
  DeletedSavedQuery = "deleted_saved_query",
  DeletedMultipleSavedQuery = "deleted_multiple_saved_query",
  EditedSavedQuery = "edited_saved_query",
  CreatedTeam = "created_team",
  DeletedTeam = "deleted_team",
  LiveQuery = "live_query",
  AppliedSpecPack = "applied_spec_pack",
  AppliedSpecPolicy = "applied_spec_policy",
  AppliedSpecSavedQuery = "applied_spec_saved_query",
  AppliedSpecTeam = "applied_spec_team",
  EditedAgentOptions = "edited_agent_options",
  UserAddedBySSO = "user_added_by_sso",
  UserLoggedIn = "user_logged_in",
  UserFailedLogin = "user_failed_login",
  UserCreated = "created_user",
  UserDeleted = "deleted_user",
  UserChangedGlobalRole = "changed_user_global_role",
  UserDeletedGlobalRole = "deleted_user_global_role",
  UserChangedTeamRole = "changed_user_team_role",
  UserDeletedTeamRole = "deleted_user_team_role",
  MdmEnrolled = "mdm_enrolled",
  MdmUnenrolled = "mdm_unenrolled",
  EditedMacosMinVersion = "edited_macos_min_version",
  ReadHostDiskEncryptionKey = "read_host_disk_encryption_key",
  CreatedMacOSProfile = "created_macos_profile",
  DeletedMacOSProfile = "deleted_macos_profile",
  EditedMacOSProfile = "edited_macos_profile",
  CreatedWindowsProfile = "created_windows_profile",
  DeletedWindowsProfile = "deleted_windows_profile",
  EditedWindowsProfile = "edited_windows_profile",
  // Note: Both "enabled_disk_encryption" and "enabled_macos_disk_encryption" display the same
  // message. The latter is deprecated in the API but it is retained here for backwards compatibility.
  EnabledDiskEncryption = "enabled_disk_encryption",
  EnabledMacDiskEncryption = "enabled_macos_disk_encryption",
  // Note: Both "disabled_disk_encryption" and "disabled_macos_disk_encryption" display the same
  // message. The latter is deprecated in the API but it is retained here for backwards compatibility.
  DisabledDiskEncryption = "disabled_disk_encryption",
  DisabledMacDiskEncryption = "disabled_macos_disk_encryption",
  AddedBootstrapPackage = "added_bootstrap_package",
  DeletedBootstrapPackage = "deleted_bootstrap_package",
  ChangedMacOSSetupAssistant = "changed_macos_setup_assistant",
  DeletedMacOSSetupAssistant = "deleted_macos_setup_assistant",
  EnabledMacOSSetupEndUserAuth = "enabled_macos_setup_end_user_auth",
  DisabledMacOSSetupEndUserAuth = "disabled_macos_setup_end_user_auth",
  TransferredHosts = "transferred_hosts",
  EnabledWindowsMdm = "enabled_windows_mdm",
  DisabledWindowsMdm = "disabled_windows_mdm",
  RanScript = "ran_script",
  AddedScript = "added_script",
  DeletedScript = "deleted_script",
  EditedScript = "edited_script",
  EditedWindowsUpdates = "edited_windows_updates",
  LockedHost = "locked_host",
  UnlockedHost = "unlocked_host",
  WipedHost = "wiped_host",
  RanMdmCommand = "ran_mdm_command",
  InstalledFleetd = "installed_fleetd",
  SetAccountConfiguration = "set_account_configuration",
  Locked = "locked",
  Unlocked = "unlocked",
  Wiped = "wiped",
}

/** This is a subset of ActivityType that are shown only for the host past activities */
export type IHostPastActivityType =
  | ActivityType.RanMdmCommand
  | ActivityType.InstalledFleetd
  | ActivityType.SetAccountConfiguration
  | ActivityType.CreatedMacOSProfile
  | ActivityType.EditedMacOSProfile
  | ActivityType.DeletedMacOSProfile
  | ActivityType.CreatedWindowsProfile
  | ActivityType.EnabledDiskEncryption
  | ActivityType.DisabledDiskEncryption
  | ActivityType.EditedMacosMinVersion
  | ActivityType.EditedWindowsUpdates
  | ActivityType.LockedHost
  | ActivityType.UnlockedHost
  | ActivityType.WipedHost
  // TODO: Check these. are these correct? figma says "locked" and "unlocked" and "wiped"
  // but we already have locked_host and unlocked_host and wiped_host
  // | ActivityType.Locked
  // | ActivityType.Unlocked
  // | ActivityType.Wiped
  | ActivityType.RanScript;

/** This is a subset of ActivityType that are shown only for the host upcoming activities */
export type IHostUpcomingActivityType =
  | ActivityType.RanMdmCommand
  | ActivityType.InstalledFleetd
  | ActivityType.SetAccountConfiguration
  | ActivityType.CreatedMacOSProfile
  | ActivityType.EditedMacOSProfile
  | ActivityType.CreatedWindowsProfile
  | ActivityType.DeletedMacOSProfile
  | ActivityType.EnabledDiskEncryption
  | ActivityType.DisabledDiskEncryption
  | ActivityType.EditedMacosMinVersion
  | ActivityType.EditedWindowsUpdates
  | ActivityType.LockedHost
  | ActivityType.UnlockedHost
  | ActivityType.WipedHost
  | ActivityType.RanScript;
// TODO: Check these. are these correct? figma says "locked" and "unlocked" and "wiped"
// but we already have locked_host and unlocked_host and wiped_host
// | ActivityType.Locked
// | ActivityType.Unlocked
// | ActivityType.Wiped

export interface IActivity {
  created_at: string;
  id: number;
  actor_full_name: string;
  actor_id?: number;
  actor_gravatar: string;
  actor_email?: string;
  type: ActivityType;
  details?: IActivityDetails;
  fleet_initiated_activity?: boolean; // TODO: move this into IPastActivity and IUpcomingActivity?
}

export type IPastActivity = Omit<IActivity, "type"> & {
  type: IHostPastActivityType;
};

export type IUpcomingActivity = Omit<IActivity, "type"> & {
  type: IHostUpcomingActivityType;
};

// typeguard to determine if an activity is a upcoming activity response
export const isUpcomingActivityResponse = (
  activities: IPastActivitiesResponse | IUpcomingActivitiesResponse
): activities is IUpcomingActivitiesResponse => {
  return "count" in activities;
};

type IDetailsType = "mdm_command" | "script";
type IMdmCommmandStatus = "Pending" | "Acknowledged" | "Failed";

export interface IActivityDetails {
  pack_id?: number;
  pack_name?: string;
  policy_id?: number;
  policy_name?: string;
  query_id?: number;
  query_name?: string;
  query_sql?: string;
  query_ids?: number[];
  team_id?: number | null;
  team_name?: string | null;
  teams?: ITeamSummary[];
  targets_count?: number;
  specs?: IQuery[] | IPolicy[];
  global?: boolean;
  public_ip?: string;
  user_id?: number;
  user_email?: string;
  email?: string;
  role?: UserRole;
  host_serial?: string;
  host_display_name?: string;
  host_display_names?: string[];
  host_ids?: number[];
  host_platform?: string;
  installed_from_dep?: boolean;
  mdm_platform?: "microsoft" | "apple";
  minimum_version?: string;
  deadline?: string;
  profile_name?: string;
  profile_identifier?: string;
  bootstrap_package_name?: string;
  name?: string;
  script_execution_id?: string;
  script_name?: string;
  deadline_days?: number;
  grace_period_days?: number;
  stats?: IScheduledQueryStats;
  host_id?: number;
  type?: IDetailsType;
  status?: IMdmCommmandStatus;
}
