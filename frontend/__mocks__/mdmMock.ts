import { IHostMdmData } from "interfaces/host";
import {
  IMdmSolution,
  IMdmProfile,
  IMdmSummaryMdmSolution,
  IMdmCommandResult,
} from "interfaces/mdm";
import { IMdmCommandResultResponse } from "services/entities/mdm";

const DEFAULT_MDM_SOLUTION_MOCK: IMdmSolution = {
  id: 1,
  name: "MDM Solution",
  server_url: "http://mdmsolution.com",
  hosts_count: 5,
};

export const createMockMdmSolution = (
  overrides?: Partial<IMdmSolution>
): IMdmSolution => {
  return { ...DEFAULT_MDM_SOLUTION_MOCK, ...overrides };
};

const DEFAULT_HOST_SUMMARY_MDM_SOLUTION_MOCK: IMdmSummaryMdmSolution = {
  id: 1,
  name: "MDM Solution",
  server_url: "http://mdmsolution.com",
  hosts_count: 5,
};

export const createMockMdmSummaryMdmSolution = (
  overrides?: Partial<IMdmSummaryMdmSolution>
): IMdmSummaryMdmSolution => {
  return { ...DEFAULT_HOST_SUMMARY_MDM_SOLUTION_MOCK, ...overrides };
};

const DEFAULT_MDM_PROFILE_DATA: IMdmProfile = {
  profile_uuid: "123-abc",
  team_id: 0,
  name: "Test Profile",
  platform: "darwin",
  identifier: "com.test.profile",
  created_at: "2021-01-01T00:00:00Z",
  updated_at: "2021-01-01T00:00:00Z",
  checksum: "123abc",
};

export const createMockMdmProfile = (
  overrides?: Partial<IMdmProfile>
): IMdmProfile => {
  return { ...DEFAULT_MDM_PROFILE_DATA, ...overrides };
};

const DEFAULT_HOST_MDM_DATA: IHostMdmData = {
  encryption_key_available: false,
  enrollment_status: "On (automatic)",
  server_url: "http://mdmsolution.com",
  name: "MDM Solution",
  id: 1,
  profiles: [],
  os_settings: {
    disk_encryption: {
      status: "verified",
      detail: "",
    },
  },
  macos_settings: {
    disk_encryption: null,
    action_required: null,
  },
  macos_setup: {
    bootstrap_package_status: "",
    details: "",
    bootstrap_package_name: "",
  },
  device_status: "unlocked",
  pending_action: "",
};

export const createMockHostMdmData = (
  overrides?: Partial<IHostMdmData>
): IHostMdmData => {
  return { ...DEFAULT_HOST_MDM_DATA, ...overrides };
};

const DEFAULT_MOCK_MDM_COMMAND_RESULT: IMdmCommandResult = {
  host_uuid: "123-abc",
  command_uuid: "456-def",
  status: "pending",
  updated_at: "2021-01-01T00:00:00Z",
  request_type: "InstallProfile",
  hostname: "My Host",
  payload: "VGhpcyBpcyBteSBwYXlsb2FkIQ==", // 64 bit encoded for "This is my payload!"
  result: "VGhpcyBpcyBteSByZXN1bHQh", // 64 bit encoded for "This is my result!"
};

export const createMockMdmCommandResult = (
  overrides?: Partial<IMdmCommandResult>
): IMdmCommandResult => {
  return { ...DEFAULT_MOCK_MDM_COMMAND_RESULT, ...overrides };
};

export const createMockMdmCommandResultResponse = (
  overrides?: Partial<IMdmCommandResultResponse>
): IMdmCommandResultResponse => {
  return {
    results: [createMockMdmCommandResult()],
    ...overrides,
  };
};
