import { http, HttpResponse } from "msw";

import { createMockHostsResponse } from "__mocks__/hostMock";
import { createMockLabel } from "__mocks__/labelsMock";
import { IHost } from "interfaces/host";
import { ILabel, ILabelSummary } from "interfaces/label";
import { baseUrl } from "test/test-utils";

export const getLabelsSummaryHandler = (labels: ILabelSummary[]) =>
  http.get(baseUrl("/labels/summary"), () => {
    return HttpResponse.json({ labels });
  });

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
