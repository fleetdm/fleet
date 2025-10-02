import { http, HttpResponse } from "msw";
import { baseUrl } from "test/test-utils";
import { createMockSoftwareInstallResult } from "__mocks__/softwareMock";

// Installed with outputs
export const getDefaultSoftwareInstallHandler = http.get(
  baseUrl("/software/install/:install_uuid/results"),
  ({ params }) => {
    return HttpResponse.json({
      results: createMockSoftwareInstallResult({
        install_uuid: params.install_uuid as string,
        status: "installed",
        output: "Install script ran",
        post_install_script_output: "Post-install success",
      }),
    });
  }
);

// Installed, no outputs
export const getSoftwareInstallHandlerNoOutputs = http.get(
  baseUrl("/software/install/:install_uuid/results"),
  ({ params }) => {
    return HttpResponse.json({
      results: createMockSoftwareInstallResult({
        install_uuid: params.install_uuid as string,
        status: "installed",
        output: "",
        post_install_script_output: "",
      }),
    });
  }
);

// Installed, only install output
export const getSoftwareInstallHandlerOnlyInstallOutput = http.get(
  baseUrl("/software/install/:install_uuid/results"),
  ({ params }) => {
    return HttpResponse.json({
      results: createMockSoftwareInstallResult({
        install_uuid: params.install_uuid as string,
        status: "installed",
        output: "Install only",
        post_install_script_output: "",
      }),
    });
  }
);
