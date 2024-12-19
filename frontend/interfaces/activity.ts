import { ILabelSoftwareTitle } from "./label";
import { Platform } from "./platform";
import { IPolicy } from "./policy";
import { IQuery } from "./query";
import { ISchedulableQueryStats } from "./schedulable_query";
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
  FleetEnrolled = "fleet_enrolled",
  MdmEnrolled = "mdm_enrolled",
  MdmUnenrolled = "mdm_unenrolled",
  EditedMacosMinVersion = "edited_macos_min_version",
  EditedIosMinVersion = "edited_ios_min_version",
  EditedIpadosMinVersion = "edited_ipados_min_version",
  ReadHostDiskEncryptionKey = "read_host_disk_encryption_key",
  /** Note: BE not renamed (yet) from macOS even though activity is also used for iOS and iPadOS */
  CreatedAppleOSProfile = "created_macos_profile",
  /** Note: BE not renamed (yet) from macOS even though activity is also used for iOS and iPadOS */
  DeletedAppleOSProfile = "deleted_macos_profile",
  /** Note: BE not renamed (yet) from macOS even though activity is also used for iOS and iPadOS */
  EditedAppleOSProfile = "edited_macos_profile",
  AddedNdesScepProxy = "added_ndes_scep_proxy",
  DeletedNdesScepProxy = "deleted_ndes_scep_proxy",
  EditedNdesScepProxy = "edited_ndes_scep_proxy",
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
  EnabledWindowsMdmMigration = "enabled_windows_mdm_migration",
  DisabledWindowsMdmMigration = "disabled_windows_mdm_migration",
  RanScript = "ran_script",
  AddedScript = "added_script",
  DeletedScript = "deleted_script",
  EditedScript = "edited_script",
  EditedWindowsUpdates = "edited_windows_updates",
  LockedHost = "locked_host",
  UnlockedHost = "unlocked_host",
  WipedHost = "wiped_host",
  CreatedDeclarationProfile = "created_declaration_profile",
  DeletedDeclarationProfile = "deleted_declaration_profile",
  EditedDeclarationProfile = "edited_declaration_profile",
  ResentConfigurationProfile = "resent_configuration_profile",
  AddedSoftware = "added_software",
  EditedSoftware = "edited_software",
  DeletedSoftware = "deleted_software",
  InstalledSoftware = "installed_software",
  UninstalledSoftware = "uninstalled_software",
  EnabledVpp = "enabled_vpp",
  DisabledVpp = "disabled_vpp",
  AddedAppStoreApp = "added_app_store_app",
  DeletedAppStoreApp = "deleted_app_store_app",
  InstalledAppStoreApp = "installed_app_store_app",
  EnabledActivityAutomations = "enabled_activity_automations",
  EditedActivityAutomations = "edited_activity_automations",
  DisabledActivityAutomations = "disabled_activity_automations",
}

// This is a subset of ActivityType that are shown only for the host past activities
export type IHostPastActivityType =
  | ActivityType.RanScript
  | ActivityType.LockedHost
  | ActivityType.UnlockedHost
  | ActivityType.InstalledSoftware
  | ActivityType.UninstalledSoftware
  | ActivityType.InstalledAppStoreApp;

// This is a subset of ActivityType that are shown only for the host upcoming activities
export type IHostUpcomingActivityType =
  | ActivityType.RanScript
  | ActivityType.InstalledSoftware
  | ActivityType.UninstalledSoftware
  | ActivityType.InstalledAppStoreApp;

export interface IActivity {
  created_at: string;
  id: number;
  actor_full_name: string;
  actor_id: number;
  actor_gravatar: string;
  actor_email?: string;
  type: ActivityType;
  details?: IActivityDetails;
}

export type IHostPastActivity = Omit<IActivity, "type" | "details"> & {
  type: IHostPastActivityType;
  details: IActivityDetails;
};

export type IHostUpcomingActivity = Omit<IActivity, "type" | "details"> & {
  type: IHostUpcomingActivityType;
  details: IActivityDetails;
};

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
  host_id?: number;
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
  stats?: ISchedulableQueryStats;
  software_title?: string;
  software_package?: string;
  platform?: Platform; // software platform
  status?: string;
  install_uuid?: string;
  self_service?: boolean;
  command_uuid?: string;
  app_store_id?: number;
  location?: string; // name of location associated with VPP token
  webhook_url?: string;
  software_title_id?: number;
  labels_include_any?: ILabelSoftwareTitle[];
  labels_exclude_any?: ILabelSoftwareTitle[];
}
