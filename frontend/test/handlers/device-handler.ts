import { http, HttpResponse } from "msw";

import createMockDeviceUser, {
  createMockDeviceSoftwareResponse,
  createMockSetupSoftwareStatusesResponse,
} from "__mocks__/deviceUserMock";
import createMockHost from "__mocks__/hostMock";
import createMockLicense from "__mocks__/licenseMock";
import createMockMacAdmins from "__mocks__/macAdminsMock";
import { createMockHostCertificate } from "__mocks__/certificatesMock";
import { createMockMdmCommandResult } from "__mocks__/mdmMock";

import { baseUrl } from "test/test-utils";
import { IDeviceUserResponse } from "interfaces/host";
import {
  IGetDeviceSoftwareResponse,
  IGetSetupExperienceStatusesResponse,
} from "services/entities/device_user";
import { IGetHostCertificatesResponse } from "services/entities/hosts";

export const defaultDeviceHandler = http.get(baseUrl("/device/:token"), () => {
  return HttpResponse.json({
    host: createMockHost(),
    license: createMockLicense(),
    org_logo_url: "",
    org_logo_url_light_background: "",
    global_config: {
      mdm: { enabled_and_configured: false },
    },
  });
});

export const customDeviceHandler = (overrides?: Partial<IDeviceUserResponse>) =>
  http.get(baseUrl("/device/:token"), () => {
    const response = Object.assign(
      {
        host: createMockHost(),
        license: createMockLicense(),
        org_logo_url: "",
        org_logo_url_light_background: "",
        global_config: {
          mdm: { enabled_and_configured: false },
        },
      },
      overrides
    );
    return HttpResponse.json(response);
  });

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

export const defaultDeviceCertificatesHandler = http.get(
  baseUrl("/device/:token/certificates"),
  () => {
    return HttpResponse.json<IGetHostCertificatesResponse>({
      certificates: [createMockHostCertificate()],
      meta: {
        has_next_results: false,
        has_previous_results: false,
      },
      count: 1,
    });
  }
);

export const deviceSetupExperienceHandler = (
  overrides?: Partial<IGetSetupExperienceStatusesResponse>
) =>
  http.post(baseUrl("/device/:token/setup_experience/status"), () => {
    return HttpResponse.json(
      createMockSetupSoftwareStatusesResponse(overrides)
    );
  });

export const emptySetupExperienceHandler = deviceSetupExperienceHandler({
  setup_experience_results: { software: [], scripts: [] },
});

export const getDeviceVppCommandResultHandler = http.get(
  `/device/:token/software/commands/:uuid/results`,
  ({ params }) => {
    const { token, uuid } = params;

    // Map UUIDs to status
    const statusMap = {
      "notnow-uuid": "NotNow",
      "acknowledged-uuid": "Acknowledged",
      "uuid-failed": "Failed",
    };
    const status =
      statusMap[uuid as "notnow-uuid" | "acknowledged-uuid" | "uuid-failed"] ||
      "Acknowledged";

    const mdmCommand = createMockMdmCommandResult({
      command_uuid: uuid as string,
      status,
      payload: btoa(`payload for ${uuid}`),
      result: btoa(`result for ${uuid}`),
    });

    return HttpResponse.json({
      results: [mdmCommand],
    });
  }
);
