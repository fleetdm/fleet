import { ISoftware } from "interfaces/software";
import {
  ISoftwareTitle,
  ISoftwareTitlesResponse,
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

const DEFAULT_SOFTWARE_TITLE_MOCK: ISoftwareTitle = {
  id: 1,
  name: "mock software 1.app",
  versions_count: 1,
  source: "apps",
  hosts_count: 1,
  versions: [
    {
      id: 1,
      version: "1.0.0",
      vulnerabilities: null,
    },
  ],
};

export const createMockSoftwareTitle = (
  overrides?: Partial<ISoftwareTitle>
): ISoftwareTitle => {
  return { ...DEFAULT_SOFTWARE_TITLE_MOCK, ...overrides };
};

const DEFAULT_SOFTWARE_TITLES_RESPONSE_MOCK: ISoftwareTitlesResponse = {
  counts_updated_at: "2020-01-01T00:00:00.000Z",
  count: 1,
  software_titles: [createMockSoftwareTitle()],
};

export const createMockSoftwareTitleReponse = (
  overrides?: Partial<ISoftwareTitlesResponse>
): ISoftwareTitlesResponse => {
  return { ...DEFAULT_SOFTWARE_TITLES_RESPONSE_MOCK, ...overrides };
};
