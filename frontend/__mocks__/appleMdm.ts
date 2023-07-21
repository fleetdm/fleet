import { IMdmApple } from "interfaces/mdm";

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

export default createMockMdmApple;
