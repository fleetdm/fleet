import { IDeviceUser } from "interfaces/host";
import { IDeviceSoftware, ISetupSoftwareStatus } from "interfaces/software";
import {
  IGetDeviceSoftwareResponse,
  IGetSetupSoftwareStatusesResponse,
} from "services/entities/device_user";
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
  icon_url: null,
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

const DEFAULT_SETUP_SOFTWARE_STATUS_MOCK: ISetupSoftwareStatus = {
  name: "Slack",
  status: "pending",
};

export const createMockSetupSoftwareStatus = (
  overrides?: Partial<ISetupSoftwareStatus>
): ISetupSoftwareStatus => {
  return { ...DEFAULT_SETUP_SOFTWARE_STATUS_MOCK, ...overrides };
};

const DEFAULT_SETUP_SOFTWARE_STATUSES_RESPONSE_MOCK: IGetSetupSoftwareStatusesResponse = {
  setup_experience_results: {
    software: [
      createMockSetupSoftwareStatus({ name: "1Password", status: "pending" }),
      createMockSetupSoftwareStatus({ name: "Chrome", status: "failure" }),
      createMockSetupSoftwareStatus({ name: "Firefox", status: "cancelled" }),
      createMockSetupSoftwareStatus({ name: "Slack", status: "success" }),
      createMockSetupSoftwareStatus({ name: "Zoom", status: "running" }),
    ],
  },
};

export const createMockSetupSoftwareStatusesResponse = (
  overrides?: Partial<IGetSetupSoftwareStatusesResponse>
): IGetSetupSoftwareStatusesResponse => {
  return {
    ...DEFAULT_SETUP_SOFTWARE_STATUSES_RESPONSE_MOCK,
    ...overrides,
  };
};

export default createMockDeviceUser;
