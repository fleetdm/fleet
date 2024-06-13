import { rest } from "msw";

import createMockDeviceUser, {
  createMockDeviceSoftwareResponse,
} from "__mocks__/deviceUserMock";
import createMockHost from "__mocks__/hostMock";
import createMockLicense from "__mocks__/licenseMock";
import createMockMacAdmins from "__mocks__/macAdminsMock";
import { baseUrl } from "test/test-utils";
import { IDeviceUserResponse } from "interfaces/host";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";

export const defaultDeviceHandler = rest.get(
  baseUrl("/device/:token"),
  (req, res, context) => {
    return res(
      context.json({
        host: createMockHost(),
        license: createMockLicense(),
        org_logo_url: "",
        global_config: {
          mdm: { enabled_and_configured: false },
        },
      })
    );
  }
);

export const customDeviceHandler = (overrides: Partial<IDeviceUserResponse>) =>
  rest.get(baseUrl("/device/:token"), (req, res, context) => {
    return res(
      context.json(
        Object.assign(
          {
            host: createMockHost(),
            license: createMockLicense(),
            org_logo_url: "",
            global_config: {
              mdm: { enabled_and_configured: false },
            },
          },
          overrides
        )
      )
    );
  });

export const defaultDeviceMappingHandler = rest.get(
  baseUrl("/device/:token/device_mapping"),
  (req, res, context) => {
    return res(
      context.json({
        device_mapping: [createMockDeviceUser()],
        host_id: 1,
      })
    );
  }
);

export const defaultMacAdminsHandler = rest.get(
  baseUrl("/device/:token/macadmins"),
  (req, res, context) => {
    return res(
      context.json({
        macadmins: createMockMacAdmins(),
      })
    );
  }
);

export const customDeviceSoftwareHandler = (
  overrides?: Partial<IGetDeviceSoftwareResponse>
) =>
  rest.get(baseUrl("/device/:token/software"), (req, res, context) => {
    return res(context.json(createMockDeviceSoftwareResponse(overrides)));
  });
