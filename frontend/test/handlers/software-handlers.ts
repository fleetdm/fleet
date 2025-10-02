import { http, HttpResponse } from "msw";
import { baseUrl } from "test/test-utils";
import { ISoftwareInstallResult } from "interfaces/software";
import { createMockSoftwareInstallResult } from "__mocks__/softwareMock";

// Installed with outputs
export const getDefaultSoftwareInstallHandler = (
  overrides: Partial<ISoftwareInstallResult>
) =>
  http.get(baseUrl("/software/install/:install_uuid/results"), ({ params }) => {
    return HttpResponse.json({
      results: createMockSoftwareInstallResult({
        install_uuid: overrides.install_uuid ?? "abc-123",
        status: "installed",
        output: "Install script ran",
        post_install_script_output: "Post-install success",
      }),
    });
  });

// Installed, no outputs
export const softwareInstallHandlerNoOutputs = http.get(
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
export const softwareInstallHandlerOnlyInstallOutput = http.get(
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
