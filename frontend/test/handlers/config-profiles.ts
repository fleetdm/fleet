import createMockConfig from "__mocks__/configMock";
import { IConfig } from "interfaces/config";
import { http, HttpResponse } from "msw";
import { baseUrl } from "test/test-utils";

const configProfileURL = baseUrl("/configuration_profiles");

export const createGetConfigHandler = (overrides?: Partial<IConfig>) => {
  return http.get(configProfileURL, () => {
    return HttpResponse.json(createMockConfig({ ...overrides }));
  });
};

export const defaultConfigProfileStatusHandler = http.get(
  `${configProfileURL}/:uuid/status`,
  () => {
    return HttpResponse.json({
      verified: 0,
      verifying: 1,
      pending: 2,
      failed: 3,
    });
  }
);
