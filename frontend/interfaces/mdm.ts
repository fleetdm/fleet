import { IConfigServerSettings } from "./config";

export interface IMdmApple {
  common_name: string;
  serial_number: string;
  issuer: string;
  renew_date: string;
}

export interface IMdmAppleBm {
  default_team?: string;
  apple_id: string;
  org_name: string;
  mdm_server_url: string;
  renew_date: string;
}

export type ITokenTeam = {
  team_id: number;
  name: string;
};

export interface IMdmAbmToken {
  id: number;
  apple_id: string;
  org_name: string;
  mdm_server_url: string;
  renew_date: string;
  terms_expired: boolean;
  macos_team: ITokenTeam;
  ios_team: ITokenTeam;
  ipados_team: ITokenTeam;
}

export interface IMdmVppToken {
  id: number;
  org_name: string;
  location: string;
  renew_date: string;
  teams: ITokenTeam[] | null; // null means token isn't configured to a team; empty array means all teams
}

export const getMdmServerUrl = ({ server_url }: IConfigServerSettings) => {
  return server_url.concat("/mdm/apple/mdm");
};

export const MDM_ENROLLMENT_STATUS = {
  "On (manual)": "manual",
  "On (automatic)": "automatic",
  Off: "unenrolled",
  Pending: "pending",
};

export type MdmEnrollmentStatus = keyof typeof MDM_ENROLLMENT_STATUS;

export interface IMdmStatusCardData {
  status: MdmEnrollmentStatus;
  hosts: number;
}

export interface IMdmAggregateStatus {
  enrolled_manual_hosts_count: number;
  enrolled_automated_hosts_count: number;
  unenrolled_hosts_count: number;
  pending_hosts_count?: number;
}

export interface IMdmSolution {
  id: number;
  name: string | null;
  server_url: string;
  hosts_count: number;
}

/** This is the mdm solution that comes back from the host/summary/mdm
request. We will always get a string for the solution name in this case  */
export interface IMdmSummaryMdmSolution extends IMdmSolution {
  name: string;
}

interface IMdmStatus {
  enrolled_manual_hosts_count: number;
  enrolled_automated_hosts_count: number;
  unenrolled_hosts_count: number;
  pending_hosts_count?: number;
  hosts_count: number;
}

export interface IMdmSummaryResponse {
  counts_updated_at: string;
  mobile_device_management_enrollment_status: IMdmStatus;
  mobile_device_management_solution: IMdmSummaryMdmSolution[] | null;
}

export type ProfilePlatform = "darwin" | "windows" | "ios" | "ipados" | "linux";

export interface IProfileLabel {
  name: string;
  id?: number; // id is only present when the label is not broken
  broken?: boolean;
}

export interface IMdmProfile {
  profile_uuid: string;
  team_id: number;
  name: string;
  platform: ProfilePlatform;
  identifier: string | null; // null for windows profiles
  created_at: string;
  updated_at: string;
  checksum: string | null; // null for windows profiles
  labels_include_all?: IProfileLabel[];
  labels_include_any?: IProfileLabel[];
  labels_exclude_any?: IProfileLabel[];
}

export type MdmProfileStatus = "verified" | "verifying" | "pending" | "failed";
export type MdmDDMProfileStatus =
  | "success"
  | "pending"
  | "failed"
  | "acknowledged";

export type ProfileOperationType = "remove" | "install";

export interface IHostMdmProfile {
  profile_uuid: string;
  name: string;
  operation_type: ProfileOperationType | null;
  platform: ProfilePlatform;
  status: MdmProfileStatus | MdmDDMProfileStatus | LinuxDiskEncryptionStatus;
  detail: string;
}

// TODO - move disk encryption related types to dedicated file
export type DiskEncryptionStatus =
  | "verified"
  | "verifying"
  | "action_required"
  | "enforcing"
  | "failed"
  | "removing_enforcement";

/** Currently windows disk enxryption status will only be one of these four
values. In the future we may add more. */
export type WindowsDiskEncryptionStatus = Extract<
  DiskEncryptionStatus,
  "verified" | "verifying" | "enforcing" | "failed"
>;

export const isWindowsDiskEncryptionStatus = (
  status: DiskEncryptionStatus
): status is WindowsDiskEncryptionStatus => {
  switch (status) {
    case "verified":
    case "verifying":
    case "enforcing":
    case "failed":
      return true;
    default:
      return false;
  }
};

export type LinuxDiskEncryptionStatus = Extract<
  DiskEncryptionStatus,
  "verified" | "failed" | "action_required"
>;

export const isLinuxDiskEncryptionStatus = (
  status: DiskEncryptionStatus
): status is LinuxDiskEncryptionStatus =>
  ["verified", "failed", "action_required"].includes(status);

export const FLEET_FILEVAULT_PROFILE_DISPLAY_NAME = "Disk encryption";

export interface IMdmSSOReponse {
  url: string;
}

export interface IBootstrapPackageMetadata {
  name: string;
  team_id: number;
  sha256: string;
  token: string;
  created_at: string;
}

export interface IBootstrapPackageAggregate {
  installed: number;
  pending: number;
  failed: number;
}

export enum BootstrapPackageStatus {
  INSTALLED = "installed",
  PENDING = "pending",
  FAILED = "failed",
}

/**
 * IMdmCommandResult is the shape of an mdm command result object
 * returned by the Fleet API.
 */
export interface IMdmCommandResult {
  host_uuid: string;
  command_uuid: string;
  /** Status is the status of the command. It can be one of Acknowledged, Error, or NotNow for
	// Apple, or 200, 400, etc for Windows.  */
  status: string;
  updated_at: string;
  request_type: string;
  hostname: string;
  /** Payload is a base64-encoded string containing the MDM command request */
  payload: string;
  /** Result is a base64-enconded string containing the MDM command response */
  result: string;
}
