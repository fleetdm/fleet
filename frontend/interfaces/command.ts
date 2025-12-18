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
}
