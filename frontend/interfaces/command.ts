export type CommandStatus = "Pending" | "Acknowledged" | "Error" | "NotNow";

export interface ICommand {
  host_uuid: string;
  command_uuid: string;
  status: string;
  updated_at: string;
  request_type: string;
  hostname: string;
}
