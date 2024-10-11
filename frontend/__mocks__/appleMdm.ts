import { IMdmApple } from "interfaces/mdm";
import { IGetVppInfoResponse, IVppApp } from "services/entities/mdm_apple";

const DEFAULT_MDM_APPLE_MOCK: IMdmApple = {
  common_name: "APSP:12345",
  serial_number: "12345",
  issuer: "Test Certification Authority",
  renew_date: "2023-03-24T22:13:59Z",
};

export const createMockMdmApple = (
  overrides?: Partial<IMdmApple>
): IMdmApple => {
  return { ...DEFAULT_MDM_APPLE_MOCK, ...overrides };
};

const DEFAULT_MDM_APPLE_VPP_INFO_MOCK: IGetVppInfoResponse = {
  org_name: "test org",
  renew_date: "2024-09-19T00:00:00Z",
  location: "test location",
};

export const createMockVppInfo = (
  overrides?: Partial<IGetVppInfoResponse>
): IGetVppInfoResponse => {
  return { ...DEFAULT_MDM_APPLE_VPP_INFO_MOCK, ...overrides };
};

const DEFAULT_MDM_APPLE_VPP_APP_MOCK: IVppApp = {
  name: "Test App",
  bundle_identifier: "com.test.app",
  icon_url: "https://via.placeholder.com/512",
  latest_version: "1.0",
  app_store_id: "1",
  added: false,
  platform: "darwin",
};

export const createMockVppApp = (overrides?: Partial<IVppApp>): IVppApp => {
  return { ...DEFAULT_MDM_APPLE_VPP_APP_MOCK, ...overrides };
};

export default createMockMdmApple;
