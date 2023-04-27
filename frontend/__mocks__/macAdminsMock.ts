import { IMacadminsResponse } from "interfaces/host";

const DEFAULT_MAC_ADMINS_MOCK: IMacadminsResponse = {
  macadmins: {
    mobile_device_management: {
      enrollment_status: "On (manual)",
      server_url: "https://kandji.com/2",
      name: "Kandji",
      id: 11,
    },
    munki: {
      version: "1.2.3",
    },
    munki_issues: [],
  },
};

const createMockMacAdmins = (
  overrides?: Partial<IMacadminsResponse>
): IMacadminsResponse => {
  return { ...DEFAULT_MAC_ADMINS_MOCK, ...overrides };
};

export default createMockMacAdmins;
