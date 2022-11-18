import { IDeviceUser } from "interfaces/host";

const DEFAULT_DEVICE_USER_MOCK: IDeviceUser = {
  email: "test@test.com",
  source: "test_source",
};

const createMockDeviceUser = (
  overrides?: Partial<IDeviceUser>
): IDeviceUser => {
  return { ...DEFAULT_DEVICE_USER_MOCK, ...overrides };
};

export default createMockDeviceUser;
