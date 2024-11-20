import {
  ISoftware,
  ISoftwareVersion,
  ISoftwareVulnerability,
  ISoftwareTitleVersion,
  ISoftwarePackage,
  ISoftwareTitle,
  ISoftwareTitleDetails,
  IAppStoreApp,
  IFleetMaintainedApp,
  IFleetMaintainedAppDetails,
} from "interfaces/software";
import {
  ISoftwareTitlesResponse,
  ISoftwareTitleResponse,
  ISoftwareVersionsResponse,
  ISoftwareVersionResponse,
} from "services/entities/software";
import { IOSVersionsResponse } from "../services/entities/operating_systems";
import { IOperatingSystemVersion } from "../interfaces/operating_system";

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
  source: "apps",
  browser: "chrome",
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

export const createMockSoftwareVersionsResponse = (
  overrides?: Partial<ISoftwareVersionsResponse>
): ISoftwareVersionsResponse => {
  return { ...DEFAULT_SOFTWARE_VERSIONS_RESPONSE_MOCK, ...overrides };
};

const DEFAULT_OS_VERSION_MOCK = {
  os_version_id: 1,
  name: "macOS 14.6.1",
  name_only: "macOS",
  version: "14.6.1",
  platform: "darwin",
  hosts_count: 42,
  generated_cpes: [],
  vulnerabilities: [],
};

export const createMockOSVersion = (
  overrides?: Partial<IOperatingSystemVersion>
): IOperatingSystemVersion => {
  return {
    ...DEFAULT_OS_VERSION_MOCK,
    ...overrides,
  };
};

const DEFAULT_OS_VERSIONS_RESPONSE_MOCK: IOSVersionsResponse = {
  counts_updated_at: "2020-01-01T00:00:00.000Z",
  count: 1,
  os_versions: [createMockOSVersion()],
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

export const createMockOSVersionsResponse = (
  overrides?: Partial<IOSVersionsResponse>
): IOSVersionsResponse => {
  return { ...DEFAULT_OS_VERSIONS_RESPONSE_MOCK, ...overrides };
};

const DEFAULT_APP_STORE_APP_MOCK: IAppStoreApp = {
  name: "test app",
  app_store_id: 1,
  icon_url: "https://via.placeholder.com/512",
  latest_version: "1.2.3",
  self_service: true,
  status: {
    installed: 1,
    pending: 2,
    failed: 3,
  },
};

export const createMockAppStoreApp = (overrides?: Partial<IAppStoreApp>) => {
  return { ...DEFAULT_APP_STORE_APP_MOCK, ...overrides };
};

const DEFAULT_SOFTWARE_TITLE_DETAILS_MOCK: ISoftwareTitleDetails = {
  id: 1,
  name: "test.app",
  software_package: null,
  app_store_app: null,
  source: "apps",
  hosts_count: 1,
  versions: [createMockSoftwareTitleVersion()],
  bundle_identifier: "com.test.Desktop",
  versions_count: 1,
  counts_updated_at: "2024-11-01T01:23:45Z",
};

export const createMockSoftwareTitleDetails = (
  overrides?: Partial<ISoftwareTitleDetails>
) => {
  return { ...DEFAULT_SOFTWARE_TITLE_DETAILS_MOCK, ...overrides };
};

const DEFAULT_SOFTWARE_TITLE_RESPONSE: ISoftwareTitleResponse = {
  software_title: createMockSoftwareTitleDetails(),
};

export const createMockSoftwareTitleResponse = (
  overrides?: Partial<ISoftwareTitleResponse>
): ISoftwareTitleResponse => {
  return { ...DEFAULT_SOFTWARE_TITLE_RESPONSE, ...overrides };
};

const DEFAULT_SOFTWARE_VERSION_RESPONSE = {
  software: createMockSoftwareVersion(),
};

export const createMockSoftwareVersionResponse = (
  overrides?: Partial<ISoftwareVersionResponse>
): ISoftwareVersionResponse => {
  return { ...DEFAULT_SOFTWARE_VERSION_RESPONSE, ...overrides };
};

const DEFAULT_SOFTWARE_PACKAGE_MOCK: ISoftwarePackage = {
  name: "TestPackage-1.2.3.pkg",
  version: "1.2.3",
  uploaded_at: "2020-01-01T00:00:00.000Z",
  install_script: "sudo installer -pkg /temp/FalconSensor-6.44.pkg -target /",
  pre_install_query: "SELECT 1 FROM macos_profiles WHERE uuid='abc123';",
  post_install_script:
    "sudo /Applications/Falcon.app/Contents/Resources/falconctl license abc123",
  self_service: false,
  icon_url: null,
  status: {
    installed: 1,
    pending_install: 2,
    failed_install: 1,
    pending_uninstall: 1,
    failed_uninstall: 1,
  },
  automatic_install_policies: [],
  last_install: null,
  last_uninstall: null,
  package_url: "",
};

export const createMockSoftwarePackage = (
  overrides?: Partial<ISoftwarePackage>
) => {
  return { ...DEFAULT_SOFTWARE_PACKAGE_MOCK, ...overrides };
};

const DEFAULT_SOFTWARE_TITLE_MOCK: ISoftwareTitle = {
  id: 1,
  name: "mock software 1.app",
  versions_count: 1,
  source: "apps",
  hosts_count: 1,
  browser: "chrome",
  versions: [createMockSoftwareTitleVersion()],
  software_package: createMockSoftwarePackage(),
  app_store_app: null,
};

export const createMockSoftwareTitle = (
  overrides?: Partial<ISoftwareTitle>
): ISoftwareTitle => {
  return {
    ...DEFAULT_SOFTWARE_TITLE_MOCK,
    ...overrides,
  };
};

const DEFAULT_SOFTWARE_TITLES_RESPONSE_MOCK: ISoftwareTitlesResponse = {
  counts_updated_at: "2020-01-01T00:00:00.000Z",
  count: 1,
  software_titles: [createMockSoftwareTitle()],
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

export const createMockSoftwareTitlesResponse = (
  overrides?: Partial<ISoftwareTitlesResponse>
): ISoftwareTitlesResponse => {
  return { ...DEFAULT_SOFTWARE_TITLES_RESPONSE_MOCK, ...overrides };
};

const DEFAULT_FLEET_MAINTAINED_APPS_MOCK: IFleetMaintainedApp = {
  id: 1,
  name: "test app",
  version: "1.2.3",
  platform: "darwin",
};

export const createMockFleetMaintainedApp = (
  overrides?: Partial<IFleetMaintainedApp>
): IFleetMaintainedApp => {
  return {
    ...DEFAULT_FLEET_MAINTAINED_APPS_MOCK,
    ...overrides,
  };
};

const DEFAULT_FLEET_MAINTAINED_APP_DETAILS_MOCK: IFleetMaintainedAppDetails = {
  id: 1,
  name: "Test app",
  version: "1.2.3",
  platform: "darwin",
  pre_install_script: "SELECT * FROM osquery_info WHERE start_time > 1",
  install_script: '#!/bin/sh\n\ninstaller -pkg "$INSTALLER" -target /',
  post_install_script: 'echo "Installed"',
  uninstall_script:
    "#!/bin/sh\n\n# Fleet extracts and saves package IDs\npkg_ids=$PACKAGE_ID",
};

export const createMockFleetMaintainedAppDetails = (
  overrides?: Partial<IFleetMaintainedAppDetails>
) => {
  return { ...DEFAULT_FLEET_MAINTAINED_APP_DETAILS_MOCK, ...overrides };
};
