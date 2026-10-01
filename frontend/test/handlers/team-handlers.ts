import { http, HttpResponse } from "msw";

import createMockConfig from "__mocks__/configMock";
import { IConfig } from "interfaces/config";
import { baseUrl } from "test/test-utils";

const teamUrl = baseUrl("/fleets/:id");

// eslint-disable-next-line import/prefer-default-export
export const createGetTeamHandler = (overrides?: Partial<IConfig>) => {
  return http.get(teamUrl, () => {
    return HttpResponse.json(createMockConfig({ ...overrides }));
  });
};
