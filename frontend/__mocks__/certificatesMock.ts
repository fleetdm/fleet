import { IHostCertificate } from "interfaces/certificates";
import { IGetHostCertificatesResponse } from "services/entities/hosts";

const DEFAULT_HOST_CERTIFICATE_MOCK: IHostCertificate = {
  id: 1,
  not_valid_after: "2021-08-19T02:02:17Z",
  not_valid_before: "2021-08-19T02:02:17Z",
  certificate_authority: true,
  common_name: "Test Cert",
  key_algorithm: "rsaEncryption",
  key_strength: 2048,
  key_usage: "CRL Sign, Key Cert Sign",
  serial: "123",
  signing_algorithm: "sha256WithRSAEncryption",
  subject: {
    country: "US",
    organization: "Test Inc.",
    organizational_unit: "Test Inc.",
    common_name: "Test Biz",
  },
  issuer: {
    country: "US",
    organization: "Test Inc.",
    organizational_unit: "Test Inc.",
    common_name: "Test Biz",
  },
  source: "system",
  username: "",
};

export const createMockHostCertificate = (
  overrides?: Partial<IHostCertificate>
): IHostCertificate => {
  return { ...DEFAULT_HOST_CERTIFICATE_MOCK, ...overrides };
};

const DEFAULT_HOST_CERTIFICATES_RESPONSE_MOCK: IGetHostCertificatesResponse = {
  certificates: [createMockHostCertificate()],
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

export const createMockGetHostCertificatesResponse = (
  overrides?: Partial<IGetHostCertificatesResponse>
): IGetHostCertificatesResponse => {
  return { ...DEFAULT_HOST_CERTIFICATES_RESPONSE_MOCK, ...overrides };
};
