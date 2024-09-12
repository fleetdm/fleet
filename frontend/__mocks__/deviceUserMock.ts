import { IDeviceUser } from "interfaces/host";
import { IDeviceSoftware } from "interfaces/software";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { createMockHostSoftwarePackage } from "./hostMock";

const DEFAULT_DEVICE_USER_MOCK: IDeviceUser = {
  email: "test@test.com",
  source: "test_source",
};

const createMockDeviceUser = (
  overrides?: Partial<IDeviceUser>
): IDeviceUser => {
  return { ...DEFAULT_DEVICE_USER_MOCK, ...overrides };
};

const DEFAULT_DEVICE_SOFTWARE_MOCK: IDeviceSoftware = {
  id: 1,
  name: "mock software 1.app",
  source: "apps",
  bundle_identifier: "com.app.mock",
  status: null,
  installed_versions: null,
  software_package: createMockHostSoftwarePackage(),
  app_store_app: null,
};

export const createMockDeviceSoftware = (
  overrides?: Partial<IDeviceSoftware>
) => {
  return { ...DEFAULT_DEVICE_SOFTWARE_MOCK, ...overrides };
};

const DEFAULT_DEVICE_SOFTWARE_RESPONSE_MOCK = {
  software: [createMockDeviceSoftware()],
  count: 0,
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

export const createMockDeviceSoftwareResponse = (
  overrides?: Partial<IGetDeviceSoftwareResponse>
) => {
  return {
    ...DEFAULT_DEVICE_SOFTWARE_RESPONSE_MOCK,
    ...overrides,
  };
};

export default createMockDeviceUser;
