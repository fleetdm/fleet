/** The original mdm command status representation */
export type CommandStatus = "Pending" | "Acknowledged" | "Error" | "NotNow";

/** The fleet representation of command status */
export type FleetCommandStatus = "ran" | "pending" | "failed";

export interface ICommand {
  host_uuid: string;
  command_uuid: string;
  status: CommandStatus;
  command_status: FleetCommandStatus;
  updated_at: string;
  request_type: string;
  hostname: string;
  name: string | null; // Profile name when command is for installing/removing a macOS profile
}

/**
 * Shape of an mdm command result object returned by the Fleet API.
 */
export interface ICommandResult {
  host_uuid: string;
  command_uuid: string;
  /** Status of the command. It can be one of Acknowledged, Error, or NotNow for
  // Apple, or 200, 400, etc for Windows.  */
  status: string;
  updated_at: string;
  request_type: string;
  hostname: string;
  /** Base64-encoded string containing the MDM command request */
  payload: string;
  /** Base64-encoded string containing the MDM command response */
  result: string;
  name: string | null; // Profile name when command is for installing/removing a macOS profile
  /** ResultsMetadata contains command-specific metadata.
   * VPP install commands include a "software_installed" boolean and
   * "vpp_verify_timeout_seconds" integer. */
  results_metadata?: Record<string, unknown>;
}
