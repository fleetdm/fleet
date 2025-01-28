import { http, HttpResponse } from "msw";

import { baseUrl } from "test/test-utils";
import { createMockLabel } from "__mocks__/labelsMock";
import { ILabel } from "interfaces/label";

// eslint-disable-next-line import/prefer-default-export
export const getLabelHandler = (overrides: Partial<ILabel>) =>
  http.get(baseUrl("/labels/:id"), () => {
    return HttpResponse.json({
      label: createMockLabel({ ...overrides }),
    });
  });
