import { rest } from "msw";

import { baseUrl } from "test/test-utils";
import { createMockSetupExperienceScript } from "__mocks__/setupExperienceMock";

const setupExperienceScriptUrl = baseUrl("/setup_experience/script");

export const defaultSetupExperienceScriptHandler = rest.get(
  setupExperienceScriptUrl,
  (req, res, context) => {
    return res(context.json(createMockSetupExperienceScript()));
  }
);

export const errorNoSetupExperienceScript = rest.get(
  setupExperienceScriptUrl,
  (req, res, context) => {
    return res(context.status(404));
  }
);
