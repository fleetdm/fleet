import { http, HttpResponse } from "msw";
import { baseUrl } from "test/test-utils";
import { createMockSoftwareInstallResult } from "__mocks__/softwareMock";
import { createMockMdmCommandResult } from "__mocks__/mdmMock";

// ---- Software Install Handlers ----

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

export const getSoftwareInstallResultHandler = http.get(
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

// ---- MDM Command Handlers ----

/** This is used for testing command results of IPA custom packages */
export const getMdmCommandResultHandler = http.get(
  baseUrl("/commands/results"),
  ({ request }) => {
    // Parse query string
    const url = new URL(request.url);
    const commandUuid = url.searchParams.get("command_uuid");

    const statusMap: Record<string, string> = {
      "notnow-uuid": "NotNow",
      "acknowledged-uuid": "Acknowledged",
    };
    const status = statusMap[commandUuid ?? ""] || "Acknowledged";

    const mdmCommand = createMockMdmCommandResult({
      command_uuid: commandUuid ?? "",
      status,
    });

    return HttpResponse.json({
      results: [mdmCommand],
    });
  }
);
