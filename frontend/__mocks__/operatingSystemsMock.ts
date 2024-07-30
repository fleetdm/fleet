import { IOperatingSystemVersion } from "interfaces/operating_system";
import { IOSVersionsResponse } from "services/entities/operating_systems";
import { createMockSoftwareVulnerability } from "./softwareMock";

const DEFAULT_OS_VERSION: IOperatingSystemVersion = {
  os_version_id: 1,
  name: "Mac OS X",
  name_only: "Mac OS X",
  version: "10.15.7",
  platform: "darwin",
  hosts_count: 1,
  generated_cpes: ["cpe:/o:apple:mac_os_x:10.15.7"],
  vulnerabilities: [createMockSoftwareVulnerability()],
};

export const createMockOSVersion = (
  overrides?: Partial<IOperatingSystemVersion>
): IOperatingSystemVersion => {
  return { ...DEFAULT_OS_VERSION, ...overrides };
};

const DEFAULT_OS_VERSIONS_RESPONSE: IOSVersionsResponse = {
  count: 1,
  counts_updated_at: "2021-01-01T00:00:00Z",
  os_versions: [createMockOSVersion()],
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

// eslint-disable-next-line import/prefer-default-export
export const createMockOSVersionsResponse = (
  overrides?: Partial<IOSVersionsResponse>
): IOSVersionsResponse => {
  return { ...DEFAULT_OS_VERSIONS_RESPONSE, ...overrides };
};
