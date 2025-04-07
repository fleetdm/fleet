import { http, HttpResponse } from "msw";

import { baseUrl } from "test/test-utils";
import { createMockSetupExperienceScript } from "__mocks__/setupExperienceMock";

const setupExperienceScriptUrl = baseUrl("/setup_experience/script");

export const defaultSetupExperienceScriptHandler = http.get(
  setupExperienceScriptUrl,
  () => {
    return HttpResponse.json(createMockSetupExperienceScript());
  }
);

export const errorNoSetupExperienceScript = http.get(
  setupExperienceScriptUrl,
  () => {
    return new HttpResponse("Not found", { status: 404 });
  }
);
