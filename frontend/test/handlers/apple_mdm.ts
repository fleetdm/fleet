import { http, HttpResponse } from "msw";

import { createMockVppInfo } from "__mocks__/appleMdm";
import { baseUrl } from "test/test-utils";

// eslint-disable-next-line import/prefer-default-export
export const defaultVppInfoHandler = http.get(baseUrl("/vpp"), () => {
  return HttpResponse.json(createMockVppInfo());
});

export const errorNoVppInfoHandler = http.get(baseUrl("/vpp"), () => {
  return new HttpResponse("Not found", { status: 404 });
});
