import { http, HttpResponse } from "msw";

import createMockDeviceUser, {
  createMockDeviceSoftwareResponse,
} from "__mocks__/deviceUserMock";
import createMockHost from "__mocks__/hostMock";
import createMockLicense from "__mocks__/licenseMock";
import createMockMacAdmins from "__mocks__/macAdminsMock";
import { baseUrl } from "test/test-utils";
import { IDeviceUserResponse } from "interfaces/host";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";

export const defaultDeviceHandler = http.get(baseUrl("/device/:token"), () => {
  return HttpResponse.json({
    host: createMockHost(),
    license: createMockLicense(),
    org_logo_url: "",
    global_config: {
      mdm: { enabled_and_configured: false },
    },
  });
});

export const customDeviceHandler = (overrides: Partial<IDeviceUserResponse>) =>
  http.get(baseUrl("/device/:token"), () => {
    return HttpResponse.json(
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
    );
  });

export const defaultDeviceMappingHandler = http.get(
  baseUrl("/device/:token/device_mapping"),
  () => {
    return HttpResponse.json({
      device_mapping: [createMockDeviceUser()],
      host_id: 1,
    });
  }
);

export const defaultMacAdminsHandler = http.get(
  baseUrl("/device/:token/macadmins"),
  () => {
    return HttpResponse.json({
      macadmins: createMockMacAdmins(),
    });
  }
);

export const customDeviceSoftwareHandler = (
  overrides?: Partial<IGetDeviceSoftwareResponse>
) =>
  http.get(baseUrl("/device/:token/software"), () => {
    return HttpResponse.json(createMockDeviceSoftwareResponse(overrides));
  });
