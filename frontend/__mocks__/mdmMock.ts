import { IHostMdmData } from "interfaces/host";
import { IMdmSolution } from "interfaces/mdm";

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

const DEFAULT_HOST_MDM_DATA: IHostMdmData = {
  encryption_key_available: false,
  enrollment_status: "On (automatic)",
  server_url: "http://mdmsolution.com",
  name: "MDM Solution",
  id: 1,
  profiles: [],
  macos_settings: {
    disk_encryption: null,
    action_required: null,
  },
};

export const createMockHostMdmData = (
  overrides?: Partial<IHostMdmData>
): IHostMdmData => {
  return { ...DEFAULT_HOST_MDM_DATA, ...overrides };
};
