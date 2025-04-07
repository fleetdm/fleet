import { http, HttpResponse } from "msw";

import { baseUrl } from "test/test-utils";
import { createMockLabel } from "__mocks__/labelsMock";
import { createMockHostsResponse } from "__mocks__/hostMock";
import { ILabel } from "interfaces/label";
import { IHost } from "interfaces/host";

// eslint-disable-next-line import/prefer-default-export
export const getLabelHandler = (overrides: Partial<ILabel>) =>
  http.get(baseUrl("/labels/:id"), () => {
    return HttpResponse.json({
      label: createMockLabel({ ...overrides }),
    });
  });

export const getLabelHostsHandler = (mockHosts: Partial<IHost>[] | undefined) =>
  http.get(baseUrl("/labels/:id/hosts"), () => {
    return HttpResponse.json(createMockHostsResponse(mockHosts));
  });
