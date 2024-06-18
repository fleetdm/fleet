import {
  ISoftware,
  ISoftwareVersion,
  ISoftwareTitleWithPackageDetail,
  ISoftwareTitleWithPackageName,
  ISoftwareVulnerability,
  ISoftwareTitleVersion,
  ISoftwarePackage,
} from "interfaces/software";
import {
  ISoftwareTitlesResponse,
  ISoftwareTitleResponse,
  ISoftwareVersionsResponse,
  ISoftwareVersionResponse,
} from "services/entities/software";

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

export const createMockSoftware = (
  overrides?: Partial<ISoftware>
): ISoftware => {
  return { ...DEFAULT_SOFTWARE_MOCK, ...overrides };
};

const DEFAULT_SOFTWARE_TITLE_VERSION_MOCK = {
  id: 1,
  version: "1.0.0",
  vulnerabilities: ["CVE-2020-0001"],
};

export const createMockSoftwareTitleVersion = (
  overrides?: Partial<ISoftwareTitleVersion>
): ISoftwareTitleVersion => {
  return { ...DEFAULT_SOFTWARE_TITLE_VERSION_MOCK, ...overrides };
};

type MockSoftwareTitle =
  | Partial<ISoftwareTitleWithPackageDetail>
  | Partial<ISoftwareTitleWithPackageName>;

const DEFAULT_SOFTWARE_TITLE_MOCK = {
  id: 1,
  name: "mock software 1.app",
  software_package: null,
  versions_count: 1,
  source: "apps",
  hosts_count: 1,
  browser: "chrome",
  versions: [createMockSoftwareTitleVersion()],
};

export const createMockSoftwareTitle = <
  T extends
    | Partial<ISoftwareTitleWithPackageDetail>
    | Partial<ISoftwareTitleWithPackageName>
>(
  overrides: T
) => {
  const mock = {
    ...DEFAULT_SOFTWARE_TITLE_MOCK,
    ...overrides,
  };
  return mock;
};

const DEFAULT_SOFTWARE_TITLES_RESPONSE_MOCK: ISoftwareTitlesResponse = {
  counts_updated_at: "2020-01-01T00:00:00.000Z",
  count: 1,
  software_titles: [
    createMockSoftwareTitle({ software_package: null, self_service: false }),
  ],
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

export const createMockSoftwareTitlesReponse = (
  overrides?: Partial<ISoftwareTitlesResponse>
): ISoftwareTitlesResponse => {
  return { ...DEFAULT_SOFTWARE_TITLES_RESPONSE_MOCK, ...overrides };
};

const DEFAULT_SOFTWARE_VULNERABILITY_MOCK = {
  cve: "CVE-2020-0001",
  details_link: "https://test.com",
  cvss_score: 9,
  epss_probability: 0.8,
  cisa_known_exploit: false,
  cve_published: "2020-01-01T00:00:00.000Z",
  cve_description: "test description",
  resolved_in_version: "1.2.3",
};

export const createMockSoftwareVulnerability = (
  overrides?: Partial<ISoftwareVulnerability>
): ISoftwareVulnerability => {
  return { ...DEFAULT_SOFTWARE_VULNERABILITY_MOCK, ...overrides };
};

const DEFAULT_SOFTWARE_VERSION_MOCK: ISoftwareVersion = {
  id: 1,
  name: "test.app",
  version: "1.2.3",
  bundle_identifier: "com.test.Desktop",
  source: "test_package",
  browser: "",
  release: "1",
  vendor: "test_vendor",
  arch: "x86_64",
  generated_cpe: "cpe:test:app:1.2.3",
  vulnerabilities: [createMockSoftwareVulnerability()],
  hosts_count: 1,
};

export const createMockSoftwareVersion = (
  overrides?: Partial<ISoftwareVersion>
): ISoftwareVersion => {
  return { ...DEFAULT_SOFTWARE_VERSION_MOCK, ...overrides };
};

const DEFAULT_SOFTWARE_VERSIONS_RESPONSE_MOCK: ISoftwareVersionsResponse = {
  counts_updated_at: "2020-01-01T00:00:00.000Z",
  count: 1,
  software: [createMockSoftwareVersion()],
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

export const createMockSoftwareVersionsReponse = (
  overrides?: Partial<ISoftwareVersionsResponse>
): ISoftwareVersionsResponse => {
  return { ...DEFAULT_SOFTWARE_VERSIONS_RESPONSE_MOCK, ...overrides };
};

const DEFAULT_SOFTWARE_TITLE_RESPONSE = {
  software_title: createMockSoftwareTitle({
    software_package: null,
  } as Partial<ISoftwareTitleWithPackageDetail>),
};

export const createMockSoftwareTitleResponse = (
  overrides: Partial<ISoftwareTitleWithPackageDetail> = {}
): ISoftwareTitleResponse => {
  const mock = DEFAULT_SOFTWARE_TITLE_RESPONSE.software_title;
  return { software_title: { ...mock, ...overrides } };
};

const DEFAULT_SOFTWARE_VERSION_RESPONSE = {
  software: createMockSoftwareVersion(),
};

export const createMockSoftwareVersionResponse = (
  overrides?: Partial<ISoftwareVersionResponse>
): ISoftwareVersionResponse => {
  return { ...DEFAULT_SOFTWARE_VERSION_RESPONSE, ...overrides };
};

const DEFAULT_SOFTWAREPACKAGE_MOCK: ISoftwarePackage = {
  name: "TestPackage-1.2.3.pkg",
  version: "1.2.3",
  uploaded_at: "2020-01-01T00:00:00.000Z",
  install_script: "sudo installer -pkg /temp/FalconSensor-6.44.pkg -target /",
  pre_install_query: "SELECT 1 FROM macos_profiles WHERE uuid='abc123';",
  post_install_script:
    "sudo /Applications/Falcon.app/Contents/Resources/falconctl license abc123",
  self_service: false,
  status: {
    installed: 1,
    pending: 2,
    failed: 3,
  },
};

export const createMockSoftwarePackage = (
  overrides?: Partial<ISoftwarePackage>
) => {
  return { ...DEFAULT_SOFTWAREPACKAGE_MOCK, ...overrides };
};
