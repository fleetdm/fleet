const DEFAULT_LICENSE_MOCK = {
  tier: "premium",
  device_count: 100,
  expiration: "2050-01-01T00:00:00Z",
  note: "test license",
  organization: "test org",
};

type License = typeof DEFAULT_LICENSE_MOCK;

const createMockLicense = (overrides?: Partial<License>): License => {
  return { ...DEFAULT_LICENSE_MOCK, ...overrides };
};

export default createMockLicense;
