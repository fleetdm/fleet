import { IMdmApple } from "interfaces/mdm";
import { IVppApp } from "services/entities/mdm_apple";

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

const DEFAULT_MDM_APPLE_VPP_APP_MOCK: IVppApp = {
  name: "Test App",
  icon_url: "https://via.placeholder.com/512",
  latest_version: "1.0",
  app_store_id: 1,
  added: false,
};

export const createMockVppApp = (overrides?: Partial<IVppApp>): IVppApp => {
  return { ...DEFAULT_MDM_APPLE_VPP_APP_MOCK, ...overrides };
};

export default createMockMdmApple;
