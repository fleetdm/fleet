import {
  IGetBootstrapPackageMetadataResponse,
  IGetBootstrapPackageSummaryResponse,
  IGetSetupExperienceScriptResponse,
  IGetSetupExperienceSoftwareResponse,
} from "services/entities/mdm";
import { createMockSoftwareTitle } from "./softwareMock";

const DEFAULT_SETUP_EXPIERENCE_SCRIPT: IGetSetupExperienceScriptResponse = {
  id: 1,
  team_id: null,
  name: "Test Script.sh",
  created_at: "2021-01-01T00:00:00Z",
  updated_at: "2021-01-01T00:00:00Z",
};

// eslint-disable-next-line import/prefer-default-export
export const createMockSetupExperienceScriptResponse = (
  overrides?: Partial<IGetSetupExperienceScriptResponse>
): IGetSetupExperienceScriptResponse => {
  return { ...DEFAULT_SETUP_EXPIERENCE_SCRIPT, ...overrides };
};

const DEFAULT_SETUP_EXPERIENCE_SOFTWARE_RESPONSE: IGetSetupExperienceSoftwareResponse = {
  counts_updated_at: "2023-10-01T00:00:00Z",
  count: 1,
  software_titles: [createMockSoftwareTitle()],
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

export const createMockSetupExperienceSoftwareResponse = (
  overrides?: Partial<IGetSetupExperienceSoftwareResponse>
): IGetSetupExperienceSoftwareResponse => {
  return { ...DEFAULT_SETUP_EXPERIENCE_SOFTWARE_RESPONSE, ...overrides };
};

const DEFAULT_BOOTSTRAP_PACKAGE_METADATA: IGetBootstrapPackageMetadataResponse = {
  name: "test-package.pkg",
  team_id: 0,
  sha256: "123abc-sha256",
  token: "123abc-token",
  created_at: "2021-01-01T00:00:00Z",
};

export const createMockBootstrapPackageMetadataResponse = (
  overrides?: Partial<IGetBootstrapPackageMetadataResponse>
): IGetBootstrapPackageMetadataResponse => {
  return { ...DEFAULT_BOOTSTRAP_PACKAGE_METADATA, ...overrides };
};

const DEFAULT_BOOTSTRAP_SUMMARY_RESPONSE: IGetBootstrapPackageSummaryResponse = {
  installed: 1,
  pending: 2,
  failed: 3,
};

export const createMockBootstrapPackageSummaryResponse = (
  overrides?: Partial<IGetBootstrapPackageSummaryResponse>
): IGetBootstrapPackageSummaryResponse => {
  return { ...DEFAULT_BOOTSTRAP_SUMMARY_RESPONSE, ...overrides };
};
