import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import {
  createCustomRenderer,
  renderWithSetup,
  baseUrl,
} from "test/test-utils";
import mockServer from "test/mock-server";
import {
  createMockHostAppStoreApp,
  createMockHostSoftware,
} from "__mocks__/hostMock";

import VppInstallDetailsModal, {
  getStatusMessage,
  ModalButtons,
} from "./VppInstallDetailsModal";

describe("getStatusMessage helper function", () => {
  it("shows NotNow message when isStatusNotNow is true", () => {
    render(
      getStatusMessage({
        displayStatus: "pending",
        isMDMStatusNotNow: true,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
      })
    );
    expect(screen.getByText(/Fleet tried to install/i)).toBeInTheDocument();
    expect(
      screen.getByText(
        /but couldn't because the host was locked or was running on battery power while in Power Nap/i
      )
    ).toBeInTheDocument();
  });

  it("shows pending acknowledged message", () => {
    render(
      getStatusMessage({
        displayStatus: "pending_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
      })
    );
    expect(
      screen.getByText(/The MDM command \(request\) to install/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /was acknowledged but the installation has not been verified/i
      )
    ).toBeInTheDocument();
    expect(screen.getByText(/Refetch/i)).toBeInTheDocument();
  });

  it("shows failed_install message for non-Apple platform when MDM command fails", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "windows",
      })
    );
    expect(
      screen.getByText(/The MDM command \(request\) to install/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Please re-attempt this installation/i)
    ).toBeInTheDocument();
  });

  it("shows Apple-specific message when MDM command fails on macOS", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "darwin",
      })
    );
    expect(screen.getByText(/The MDM command to install/i)).toBeInTheDocument();
    expect(screen.getByText(/Logic Pro/i)).toBeInTheDocument();
    expect(
      screen.getByText(/failed\. Please try again\./i)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/Please re-attempt this installation/i)
    ).not.toBeInTheDocument();
  });

  it("shows Apple-specific message when MDM command fails on iOS", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Slack",
        hostDisplayName: "Marko's iPhone",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "ios",
      })
    );
    expect(screen.getByText(/The MDM command to install/i)).toBeInTheDocument();
    expect(screen.getByText(/Slack/i)).toBeInTheDocument();
    expect(
      screen.getByText(/failed\. Please try again\./i)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/Please re-attempt this installation/i)
    ).not.toBeInTheDocument();
  });

  it("shows Apple-specific message when MDM command fails on iPadOS", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Pages",
        hostDisplayName: "Marko's iPad",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "ipados",
      })
    );
    expect(screen.getByText(/The MDM command to install/i)).toBeInTheDocument();
    expect(screen.getByText(/Pages/i)).toBeInTheDocument();
    expect(
      screen.getByText(/failed\. Please try again\./i)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/Please re-attempt this installation/i)
    ).not.toBeInTheDocument();
  });

  it("shows failed verification message for non-Apple platforms", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "windows",
      })
    );
    expect(
      screen.getByText(/installation has not been verified/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Please re-attempt this installation/i)
    ).toBeInTheDocument();
    expect(screen.queryByText(/within 10 minutes/i)).not.toBeInTheDocument();
  });

  it("shows Apple-specific failed verification message for macOS", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "darwin",
        vppVerifyTimeoutSeconds: 1200,
      })
    );
    expect(
      screen.getByText(/The host acknowledged the MDM command to install/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /the install took longer than 20 minutes, so Fleet marked it as failed/i
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/but the app failed to install/i)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/If you're updating the app and the app is open/i)
    ).not.toBeInTheDocument();
  });

  it("shows Apple-specific failed verification message for iOS", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Slack",
        hostDisplayName: "Marko's iPhone",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "ios",
      })
    );
    expect(
      screen.getByText(/The host acknowledged the MDM command to install/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /the install took longer than .*, so Fleet marked it as failed/i
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/but the app failed to install/i)
    ).not.toBeInTheDocument();
  });

  it("shows Apple-specific failed verification message for iPadOS", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Pages",
        hostDisplayName: "Marko's iPad",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "ipados",
      })
    );
    expect(
      screen.getByText(/The host acknowledged the MDM command to install/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /the install took longer than .*, so Fleet marked it as failed/i
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/but the app failed to install/i)
    ).not.toBeInTheDocument();
  });

  it("on the My device page, timeout copy omits the host display name", () => {
    render(
      getStatusMessage({
        isMyDevicePage: true,
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Keynote",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "darwin",
        vppVerifyTimeoutSeconds: 1200,
      })
    );

    expect(
      screen.getByText(
        /the install took longer than 20 minutes, so Fleet marked it as failed/i
      )
    ).toBeInTheDocument();
    expect(screen.queryByText(/Marko's MacBook Pro/i)).not.toBeInTheDocument();
  });

  it("treats failed_install as installed when app is already installed on macOS", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "darwin",
        canOverrideFailureWithInstalled: true,
        hasInstalledVersionsOnHost: true,
      })
    );

    expect(screen.getByText(/Logic Pro/i)).toBeInTheDocument();
    expect(screen.getByText(/is installed\./i)).toBeInTheDocument();
    expect(
      screen.queryByText(/If you're updating the app and the app is open/i)
    ).not.toBeInTheDocument();
  });

  it("treats failed_install as installed when app is already installed on iOS", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Slack",
        hostDisplayName: "Marko's iPhone",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "ios",
        canOverrideFailureWithInstalled: true,
        hasInstalledVersionsOnHost: true,
      })
    );

    expect(screen.getByText(/Slack/i)).toBeInTheDocument();
    expect(screen.getByText(/is installed\./i)).toBeInTheDocument();
    expect(
      screen.queryByText(/The host acknowledged the MDM command to install/i)
    ).not.toBeInTheDocument();
  });

  it("treats failed_install as installed when app is already installed on iPadOS", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Slack",
        hostDisplayName: "Marko's iPad",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "ipados",
        canOverrideFailureWithInstalled: true,
        hasInstalledVersionsOnHost: true,
      })
    );

    expect(screen.getByText(/Slack/i)).toBeInTheDocument();
    expect(screen.getByText(/is installed\./i)).toBeInTheDocument();
    expect(
      screen.queryByText(/The host acknowledged the MDM command to install/i)
    ).not.toBeInTheDocument();
  });

  it("hides macOS update tip when failed_install and app is already installed", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
        platform: "darwin",
        canOverrideFailureWithInstalled: true,
        hasInstalledVersionsOnHost: true,
      })
    );

    expect(
      screen.queryByText(/The host acknowledged the MDM command to install/i)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/but the app failed to install/i)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/If you're updating the app and the app is open/i)
    ).not.toBeInTheDocument();
  });

  it("shows pending install on host when it comes online", () => {
    render(
      getStatusMessage({
        displayStatus: "pending_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
      })
    );
    expect(screen.getByText(/when it comes online/i)).toBeInTheDocument();
  });

  it("shows default message for installed status", () => {
    render(
      getStatusMessage({
        displayStatus: "installed",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
      })
    );
    expect(screen.getByText(/Fleet installed/i)).toBeInTheDocument();
    expect(screen.getByText(/Logic Pro/i)).toBeInTheDocument();
    expect(screen.getByText(/Marko's MacBook Pro/i)).toBeInTheDocument();
  });

  it("shows manual install message when installed not through Fleet", () => {
    render(
      getStatusMessage({
        displayStatus: "installed",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "", // <-- empty
      })
    );

    expect(screen.getByText(/Logic Pro/i)).toBeInTheDocument();
    expect(screen.getByText(/is installed\./i)).toBeInTheDocument();
  });

  it("shows default message with 'the host' if host_display_name is empty", () => {
    render(
      getStatusMessage({
        displayStatus: "installed",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
      })
    );
    expect(screen.getByText(/Fleet installed/i)).toBeInTheDocument();
    expect(screen.getByText(/the host/i)).toBeInTheDocument();
  });

  it("shows relative timestamp when available", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: new Date().toISOString(),
      })
    );
    expect(screen.getByText(/\(.*ago\)/i)).toBeInTheDocument();
  });

  it("on the device user page, does not show host info", () => {
    render(
      getStatusMessage({
        isMyDevicePage: true,
        displayStatus: "installed",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
      })
    );
    expect(screen.queryByText(/Marko's MacBook Pro/i)).not.toBeInTheDocument();
  });
});

describe("VPP Install Details Modal - ModalButtons component", () => {
  it("renders Close button by default", async () => {
    const onCancel = jest.fn();

    const { user } = renderWithSetup(
      <ModalButtons displayStatus="installed" onCancel={onCancel} />
    );

    const closeButton = screen.getByRole("button", { name: /close/i });
    expect(closeButton).toBeInTheDocument();

    await user.click(closeButton);
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("renders Cancel + Retry when failed_install with deviceAuthToken", async () => {
    const onCancel = jest.fn();
    const onRetry = jest.fn();

    const { user } = renderWithSetup(
      <ModalButtons
        displayStatus="failed_install"
        deviceAuthToken="fake_token"
        onCancel={onCancel}
        onRetry={onRetry}
        hostSoftwareId={123}
      />
    );

    const cancelButton = screen.getByRole("button", { name: /cancel/i });
    const retryButton = screen.getByRole("button", { name: /retry/i });

    expect(cancelButton).toBeInTheDocument();
    expect(retryButton).toBeInTheDocument();

    // Retry should trigger onRetry + onCancel
    await user.click(retryButton);
    expect(onRetry).toHaveBeenCalledWith(123);
    expect(onCancel).toHaveBeenCalled();
  });
});

describe("VPP Install Details Modal", () => {
  const renderWithBackend = createCustomRenderer({ withBackendMock: true });

  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("renders timeout follow-up copy on host details", async () => {
    mockServer.use(
      http.get(baseUrl("/commands/results"), ({ request }) => {
        const url = new URL(request.url);
        const commandUuid = url.searchParams.get("command_uuid");

        return HttpResponse.json({
          results: [
            {
              host_uuid: "11111111-2222-3333-4444-555555555555",
              command_uuid: commandUuid,
              status: "Acknowledged",
              updated_at: "2025-08-10T12:05:00Z",
              request_type: "InstallApplication",
              hostname: "Mock iPhone",
              payload: btoa("<Command />"),
              result: btoa("<Result />"),
              results_metadata: {
                software_installed: false,
                vpp_verify_timeout_seconds: 1200,
              },
            },
          ],
        });
      })
    );

    const hostSoftware = createMockHostSoftware({
      id: 123,
      status: "failed_install",
      name: "Keynote",
      display_name: "Keynote",
      installed_versions: [],
      source: "apps",
      app_store_app: createMockHostAppStoreApp({
        platform: "darwin",
        last_install: {
          command_uuid: "acknowledged-uuid",
          installed_at: "2025-08-10T12:00:00Z",
        },
      }),
    });

    renderWithBackend(
      <VppInstallDetailsModal
        details={{
          fleetInstallStatus: "failed_install",
          hostDisplayName: "Marko's MacBook Pro",
          appName: "Keynote",
          commandUuid: "acknowledged-uuid",
          platform: "darwin",
        }}
        hostSoftware={hostSoftware}
        onCancel={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(
          /the install took longer than 20 minutes, so Fleet marked it as failed/i
        )
      ).toBeInTheDocument();
    });
    expect(
      screen.getByText(
        /If the install finishes later, Fleet will update the status when the host is refetched/i
      )
    ).toBeInTheDocument();
  });
});
