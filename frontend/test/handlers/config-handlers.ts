import createMockConfig from "__mocks__/configMock";
import { IConfig } from "interfaces/config";
import { http, HttpResponse } from "msw";
import { baseUrl } from "test/test-utils";

const configUrl = baseUrl("/config");

 
export const createGetConfigHandler = (overrides?: Partial<IConfig>) => {
  return http.get(configUrl, () => {
    return HttpResponse.json(createMockConfig({ ...overrides }));
  });
};
