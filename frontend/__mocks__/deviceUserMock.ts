import { IDeviceUser } from "interfaces/host";
import { IDeviceSoftware } from "interfaces/software";
import { ISetupStep } from "interfaces/setup";
import {
  IGetDeviceSoftwareResponse,
  IGetSetupExperienceStatusesResponse,
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

const DEFAULT_SETUP_STEP_STATUS_MOCK: ISetupStep = {
  name: "Slack",
  status: "pending",
  type: "software_install",
};

export const createMockSetupStepStatus = (
  overrides?: Partial<ISetupStep>
): ISetupStep => {
  return { ...DEFAULT_SETUP_STEP_STATUS_MOCK, ...overrides };
};

const DEFAULT_SETUP_SOFTWARE_STATUSES_RESPONSE_MOCK: IGetSetupExperienceStatusesResponse = {
  setup_experience_results: {
    software: [
      createMockSetupStepStatus({ name: "1Password", status: "pending" }),
      createMockSetupStepStatus({ name: "Chrome", status: "failure" }),
      createMockSetupStepStatus({ name: "Firefox", status: "cancelled" }),
      createMockSetupStepStatus({ name: "Slack", status: "success" }),
      createMockSetupStepStatus({ name: "Zoom", status: "running" }),
    ],
    scripts: [
      createMockSetupStepStatus({ name: "test.sh", status: "running" }),
    ],
  },
};

export const createMockSetupSoftwareStatusesResponse = (
  overrides?: Partial<IGetSetupExperienceStatusesResponse>
): IGetSetupExperienceStatusesResponse => {
  return {
    ...DEFAULT_SETUP_SOFTWARE_STATUSES_RESPONSE_MOCK,
    ...overrides,
  };
};

export default createMockDeviceUser;
