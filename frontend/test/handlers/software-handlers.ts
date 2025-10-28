import { http, HttpResponse } from "msw";
import { baseUrl } from "test/test-utils";
import { createMockSoftwareInstallResult } from "__mocks__/softwareMock";
import { createMockMdmCommandResult } from "__mocks__/mdmMock";

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

/**
 * Generic handler for /software/install/:install_uuid/results
 * Returns either a 'SoftwareInstallResult' or an MdmCommandResult[]
 * depending on the install_uuid/command_uuid supplied.
 */
export const getUniversalSoftwareInstallHandler = http.get(
  baseUrl("/software/install/:install_uuid/results"),
  ({ params }) => {
    const installUuid = params.install_uuid as string;

    if (
      installUuid.startsWith("mdm-") ||
      installUuid === "notnow-uuid" ||
      installUuid === "acknowledged-uuid"
    ) {
      const statusMap: Record<string, string> = {
        "notnow-uuid": "NotNow",
        "acknowledged-uuid": "Acknowledged",
      };

      const status = statusMap[installUuid] || "Acknowledged";

      const mdmCommand = createMockMdmCommandResult({
        command_uuid: installUuid,
        status,
      });

      // Return what Fleet API actually returns
      return HttpResponse.json({
        results: mdmCommand,
      });
    }

    // Normal fleet install
    return HttpResponse.json({
      results: createMockSoftwareInstallResult({
        install_uuid: installUuid,
        status: "installed",
        output: "Install script ran",
        post_install_script_output: "Post-install success",
      }),
    });
  }
);
