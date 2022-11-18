import { ISoftware } from "interfaces/software";

const DEFAULT_SOFTWARE_MOCK: ISoftware = {
  hosts_count: 1,
  id: 1,
  name: "mock software 1.app",
  version: "1.0.0",
  source: "apps",
  generated_cpe: "",
  vulnerabilities: null,
  last_opened_at: null,
  bundle_identifier: "com.app.mock",
};

const createMockSoftware = (overrides?: Partial<ISoftware>): ISoftware => {
  return { ...DEFAULT_SOFTWARE_MOCK, ...overrides };
};

export default createMockSoftware;
