import createMockConfig from "__mocks__/configMock";
import { IConfig } from "interfaces/config";
import { http, HttpResponse } from "msw";
import { baseUrl } from "test/test-utils";

const teamUrl = baseUrl("/teams/:id");

 
export const createGetTeamHandler = (overrides?: Partial<IConfig>) => {
  return http.get(teamUrl, () => {
    return HttpResponse.json(createMockConfig({ ...overrides }));
  });
};
