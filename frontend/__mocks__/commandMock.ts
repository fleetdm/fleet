import { ICommand, ICommandResult } from "interfaces/command";
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
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

export const createMockGetCommandsResponse = (
  overrides?: Partial<IGetCommandsResponse>
): IGetCommandsResponse => ({
  ...DEFAULT_GET_COMMANDS_RESPONSE_MOCK,
  ...overrides,
});

const DEFAULT_COMMAND_RESULT_MOCK: ICommandResult = {
  host_uuid: "11111111-2222-3333-4444-555555555555",
  command_uuid: "mock-command-uuid-1234",
  status: "Acknowledged", // or "Error", "NotNow", "200", etc.
  updated_at: "2025-08-10T12:05:00Z",
  request_type: "InstallApplication",
  hostname: "Mock iPhone",
  payload: btoa(`<Command>
    <RequestType>InstallApplication</RequestType>
    <Identifier>com.example.MockApp</Identifier>
  </Command>`),
  result: btoa(`<Result>
    <Status>Acknowledged</Status>
    <Message>Installation complete</Message>
  </Result>`),
};

/**
 * Creates mock of an Apple MDM command result that implements the ICommandResult interface.
 */

export const createMockAppleMdmCommandResult = (
  overrides?: Partial<ICommandResult>
): ICommandResult => {
  return {
    ...DEFAULT_COMMAND_RESULT_MOCK,
    ...overrides,
  };
};
