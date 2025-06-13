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
  AppliedSpecPack = "applied_spec_pack", // fleetctl
  AppliedSpecPolicy = "applied_spec_policy", // fleetctl
  AppliedSpecSavedQuery = "applied_spec_saved_query", // fleetctl
  AppliedSpecSoftware = "applied_spec_software", // fleetctl
  AppliedSpecTeam = "applied_spec_team", // fleetctl
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
  AddedDigicert = "added_digicert",
  DeletedDigicert = "deleted_digicert",
  EditedDigicert = "edited_digicert",
  AddedCustomScepProxy = "added_custom_scep_proxy",
  DeletedCustomScepProxy = "deleted_custom_scep_proxy",
  EditedCustomScepProxy = "edited_custom_scep_proxy",
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
  EnabledGitOpsMode = "enabled_gitops_mode",
  DisabledGitOpsMode = "disabled_gitops_mode",
  EnabledWindowsMdmMigration = "enabled_windows_mdm_migration",
  DisabledWindowsMdmMigration = "disabled_windows_mdm_migration",
  RanScript = "ran_script",
  RanScriptBatch = "ran_script_batch",
  AddedScript = "added_script",
  UpdatedScript = "updated_script",
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
  ResentConfigurationProfileBatch = "resent_configuration_profile_batch",
  AddedSoftware = "added_software",
  EditedSoftware = "edited_software",
  DeletedSoftware = "deleted_software",
  InstalledSoftware = "installed_software",
  UninstalledSoftware = "uninstalled_software",
  EnabledVpp = "enabled_vpp",
  DisabledVpp = "disabled_vpp",
  AddedAppStoreApp = "added_app_store_app",
  EditedAppStoreApp = "edited_app_store_app",
  DeletedAppStoreApp = "deleted_app_store_app",
  InstalledAppStoreApp = "installed_app_store_app",
  EnabledActivityAutomations = "enabled_activity_automations",
  EditedActivityAutomations = "edited_activity_automations",
  DisabledActivityAutomations = "disabled_activity_automations",
  CanceledRunScript = "canceled_run_script",
  CanceledInstallAppStoreApp = "canceled_install_app_store_app",
  CanceledInstallSoftware = "canceled_install_software",
  CanceledUninstallSoftware = "canceled_uninstall_software",
  EnabledAndroidMdm = "enabled_android_mdm",
  DisabledAndroidMdm = "disabled_android_mdm",
  ConfiguredMSEntraConditionalAccess = "added_conditional_access_integration_microsoft",
  DeletedMSEntraConditionalAccess = "deleted_conditional_access_integration_microsoft",
  // enable/disable above feature for a team
  EnabledConditionalAccessAutomations = "enabled_conditional_access_automations",
  DisabledConditionalAccessAutomations = "disabled_conditional_access_automations",
}

/** This is a subset of ActivityType that are shown only for the host past activities */
export type IHostPastActivityType =
  | ActivityType.RanScript
  | ActivityType.LockedHost
  | ActivityType.WipedHost
  | ActivityType.ReadHostDiskEncryptionKey
  | ActivityType.UnlockedHost
  | ActivityType.InstalledSoftware
  | ActivityType.UninstalledSoftware
  | ActivityType.InstalledAppStoreApp
  | ActivityType.CanceledRunScript
  | ActivityType.CanceledInstallAppStoreApp
  | ActivityType.CanceledInstallSoftware
  | ActivityType.CanceledUninstallSoftware;

/** This is a subset of ActivityType that are shown only for the host upcoming activities */
export type IHostUpcomingActivityType =
  | ActivityType.RanScript
  | ActivityType.InstalledSoftware
  | ActivityType.UninstalledSoftware
  | ActivityType.InstalledAppStoreApp;

export interface IActivity {
  created_at: string;
  id: number;
  actor_full_name?: string; // Undefined if fleet initiated / self-service
  actor_id?: number; // Undefined if fleet initiated / self-service
  actor_gravatar?: string; // Undefined if fleet initiated / self-service
  actor_email?: string;
  type: ActivityType;
  fleet_initiated: boolean;
  details?: IActivityDetails;
}

export type IHostPastActivity = Omit<IActivity, "type" | "details"> & {
  type: IHostPastActivityType;
  details: IActivityDetails;
};

export type IHostUpcomingActivity = Omit<
  IActivity,
  "id" | "type" | "details"
> & {
  uuid: string;
  type: IHostUpcomingActivityType;
  details: IActivityDetails;
};

export interface IActivityDetails {
  /** Useful for passing this data into an activity details modal */
  created_at?: string;
  app_store_id?: number;
  bootstrap_package_name?: string;
  batch_execution_id?: string;
  command_uuid?: string;
  deadline_days?: number;
  deadline?: string;
  email?: string;
  global?: boolean;
  grace_period_days?: number;
  host_display_name?: string;
  host_display_names?: string[];
  host_id?: number;
  host_ids?: number[];
  host_count?: number;
  host_platform?: string;
  host_serial?: string;
  install_uuid?: string;
  installed_from_dep?: boolean;
  labels_exclude_any?: ILabelSoftwareTitle[];
  labels_include_any?: ILabelSoftwareTitle[];
  location?: string; // name of location associated with VPP token
  mdm_platform?: "microsoft" | "apple";
  minimum_version?: string;
  name?: string;
  pack_id?: number;
  pack_name?: string;
  platform?: Platform; // software platform
  policy_id?: number;
  policy_name?: string;
  profile_identifier?: string;
  profile_name?: string;
  public_ip?: string;
  query_id?: number;
  query_ids?: number[];
  query_name?: string;
  query_sql?: string;
  role?: UserRole;
  script_execution_id?: string;
  script_name?: string;
  self_service?: boolean;
  software_package?: string;
  software_title_id?: number;
  software_title?: string;
  specs?: IQuery[] | IPolicy[];
  stats?: ISchedulableQueryStats;
  status?: string;
  targets_count?: number;
  team_id?: number | null;
  team_name?: string | null;
  teams?: ITeamSummary[];
  user_email?: string;
  user_id?: number;
  webhook_url?: string;
}
