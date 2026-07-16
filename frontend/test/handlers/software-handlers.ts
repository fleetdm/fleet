import { http, HttpResponse } from "msw";
import { baseUrl } from "test/test-utils";
import { createMockSoftwareInstallResult } from "__mocks__/softwareMock";
import { createMockAppleMdmCommandResult } from "__mocks__/commandMock";

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

// Requires Fleet Premium license (server returns 402)
export const getSoftwareInstallResultHandlerPremiumRequired = http.get(
  baseUrl("/software/install/:install_uuid/results"),
  () => {
    return HttpResponse.json(
      { message: "Requires Fleet Premium license" },
      { status: 402 }
    );
  }
);

// ---- Pre install query output ----

// Installed, outputs for pre-install, install, and post-install
export const getSoftwareInstallHandlerWithPreInstall = http.get(
  baseUrl("/software/install/:install_uuid/results"),
  ({ params }) => {
    return HttpResponse.json({
      results: {
        ...createMockSoftwareInstallResult({
          install_uuid: params.install_uuid as string,
          status: "installed",
          output: "Install script ran",
          post_install_script_output: "Post-install success",
          pre_install_query_output: "Pre-install check passed",
        }),
      },
    });
  }
);

// Failed install, only pre-install output
export const getSoftwareInstallHandlerOnlyPreInstallOutput = http.get(
  baseUrl("/software/install/:install_uuid/results"),
  ({ params }) => {
    return HttpResponse.json({
      results: {
        ...createMockSoftwareInstallResult({
          install_uuid: params.install_uuid as string,
          status: "failed_install",
          output: "",
          post_install_script_output: "",
          pre_install_query_output: "Pre-install only",
        }),
      },
    });
  }
);

// Installed, with SHA-256 hash
export const getSoftwareInstallHandlerWithHash = http.get(
  baseUrl("/software/install/:install_uuid/results"),
  ({ params }) => {
    return HttpResponse.json({
      results: createMockSoftwareInstallResult({
        install_uuid: params.install_uuid as string,
        status: "installed",
        hash_sha256:
          "e6ddb2dd089ecea38ab73ed12812df269f1447e750cf4355703340bb8aa1ad",
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

    const mdmCommand = createMockAppleMdmCommandResult({
      command_uuid: commandUuid ?? "",
      status,
    });

    return HttpResponse.json({
      results: [mdmCommand],
    });
  }
);
