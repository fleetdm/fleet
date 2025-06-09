import { http, HttpResponse } from "msw";

import { baseUrl } from "test/test-utils";
import {
  createMockBootstrapPackageMetadataResponse,
  createMockBootstrapPackageSummaryResponse,
  createMockSetupExperienceScriptResponse,
  createMockSetupExperienceSoftwareResponse,
} from "__mocks__/setupExperienceMock";
import {
  IGetBootstrapPackageMetadataResponse,
  IGetBootstrapPackageSummaryResponse,
  IGetSetupExperienceScriptResponse,
  IGetSetupExperienceSoftwareResponse,
} from "services/entities/mdm";

const setupExperienceScriptUrl = baseUrl("/setup_experience/script");
const setupExperienceInstallSoftwareUrl = baseUrl("/setup_experience/software");
const setupExperienceBootstrapMetadataUrl = baseUrl(
  "/mdm/bootstrap/:teamId/metadata"
);
const setupExperienceBootstrapSummaryUrl = baseUrl("/mdm/bootstrap/summary");

export const defaultSetupExperienceScriptHandler = http.get(
  setupExperienceScriptUrl,
  () => {
    return HttpResponse.json(createMockSetupExperienceScriptResponse());
  }
);

export const createSetupExperienceScriptHandler = (
  overrides?: Partial<IGetSetupExperienceScriptResponse>
) =>
  http.get(setupExperienceScriptUrl, () => {
    return HttpResponse.json(
      createMockSetupExperienceScriptResponse({ ...overrides })
    );
  });

export const errorNoSetupExperienceScriptHandler = http.get(
  setupExperienceScriptUrl,
  () => {
    return new HttpResponse("Not found", { status: 404 });
  }
);

export const createSetupExperienceSoftwareHandler = (
  overrides?: Partial<IGetSetupExperienceSoftwareResponse>
) =>
  http.get(setupExperienceInstallSoftwareUrl, () => {
    return HttpResponse.json(
      createMockSetupExperienceSoftwareResponse({ ...overrides })
    );
  });

export const createSetupExperienceBootstrapPackageHandler = (
  overrides?: Partial<IGetBootstrapPackageMetadataResponse>
) =>
  http.get(setupExperienceBootstrapMetadataUrl, () => {
    return HttpResponse.json(
      createMockBootstrapPackageMetadataResponse({ ...overrides })
    );
  });

export const errorNoBootstrapPackageMetadataHandler = http.get(
  setupExperienceBootstrapMetadataUrl,
  () => {
    return new HttpResponse("Not found", { status: 404 });
  }
);

export const createSetuUpExperienceBootstrapSummaryHandler = (
  overrides?: Partial<IGetBootstrapPackageSummaryResponse>
) => {
  return http.get(setupExperienceBootstrapSummaryUrl, () => {
    return HttpResponse.json(
      createMockBootstrapPackageSummaryResponse({
        ...overrides,
      })
    );
  });
};
