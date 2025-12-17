import { ICommand } from "interfaces/command";
import { IGetCommandsResponse } from "services/entities/command";

const DEFAULT_COMMAND_MOCK: ICommand = {
  host_uuid: "default-host-uuid",
  command_uuid: "default-command-uuid",
  command_status: "pending",
  status: "Pending",
  updated_at: "2024-01-01T00:00:00Z",
  request_type: "InstallProfile",
  hostname: "default-hostname",
};

export const createMockCommand = (overrides?: Partial<ICommand>): ICommand => ({
  ...DEFAULT_COMMAND_MOCK,
  ...overrides,
});

const DEFAULT_GET_COMMANDS_RESPONSE_MOCK: IGetCommandsResponse = {
  count: 1,
  results: [DEFAULT_COMMAND_MOCK],
};

export const createMockGetCommandsResponse = (
  overrides?: Partial<IGetCommandsResponse>
): IGetCommandsResponse => ({
  ...DEFAULT_GET_COMMANDS_RESPONSE_MOCK,
  ...overrides,
});
