import { rest } from "msw";

import createMockDeviceUser from "__mocks__/deviceUserMock";
import createMockHost from "__mocks__/hostMock";
import createMockLicense from "__mocks__/licenseMock";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const handlers = [
  rest.get(baseUrl("/device/:token"), (req, res, context) => {
    return res(
      context.json({
        host: createMockHost(),
        license: createMockLicense(),
        org_logo_url: "",
      })
    );
  }),

  rest.get(baseUrl("/device/:token/device_mapping"), (req, res, context) => {
    return res(
      context.json({
        device_mapping: [createMockDeviceUser()],
        host_id: 1,
      })
    );
  }),
];

export default handlers;
