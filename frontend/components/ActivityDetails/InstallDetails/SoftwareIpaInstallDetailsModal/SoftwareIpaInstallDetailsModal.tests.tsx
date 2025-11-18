import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import { getMdmCommandResultHandler } from "test/handlers/software-handlers";
import { getDeviceVppCommandResultHandler } from "test/handlers/device-handler";
import { createMockHostSoftware } from "__mocks__/hostMock";

import SoftwareIpaInstallDetailsModal from "./SoftwareIpaInstallDetailsModal";

/**
 * Helper for rendering a pre-wired modal component
 */
const renderModal = (
  overrides?: Partial<
    React.ComponentProps<typeof SoftwareIpaInstallDetailsModal>
  >
) => {
  const render = createCustomRenderer({ withBackendMock: true });
  return render(
    <SoftwareIpaInstallDetailsModal
      details={{
        fleetInstallStatus: "pending_install",
        hostDisplayName: "Marko's MacBook Pro",
        appName: "Logic Pro",
        commandUuid: "uuid-installed",
        ...overrides?.details,
      }}
      onCancel={jest.fn()}
      {...overrides}
    />
  );
};

describe("SoftwareIpaInstallDetailsModal component", () => {
  beforeEach(() => {
    mockServer.use(
      getMdmCommandResultHandler,
      getDeviceVppCommandResultHandler
    );
  });

  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("renders NotNow message for an MDM result", async () => {
    renderModal({
      details: {
        commandUuid: "notnow-uuid",
        fleetInstallStatus: "pending_install",
        hostDisplayName: "Marko's MacBook Pro",
        appName: "Logic Pro",
      },
    });

    await waitFor(() => {
      expect(screen.getByText(/Fleet tried to install/i)).toBeInTheDocument();
    });
    expect(
      screen.getByText(
        /because the host was locked or was running on battery power while in Power Nap/i
      )
    ).toBeInTheDocument();
  });

  it("renders Acknowledged pending message", async () => {
    renderModal({
      details: {
        commandUuid: "acknowledged-uuid",
        fleetInstallStatus: "pending_install",
        hostDisplayName: "Marko's MacBook Pro",
        appName: "Logic Pro",
      },
    });

    await waitFor(() => {
      expect(
        screen.getByText(
          /was acknowledged but the installation has not been verified/i
        )
      ).toBeInTheDocument();
    });
    expect(screen.getByText(/Refetch/i)).toBeInTheDocument();
  });

  it("renders normal software install status for non-MDM case", async () => {
    renderModal({
      details: {
        commandUuid: "uuid-installed",
        fleetInstallStatus: "installed",
        hostDisplayName: "Marko's MacBook Pro",
        appName: "Logic Pro",
      },
    });

    await waitFor(() => {
      expect(screen.getByText(/Fleet installed/i)).toBeInTheDocument();
    });
    expect(screen.getByText(/Logic Pro/i)).toBeInTheDocument();
    expect(screen.getByText(/Marko's MacBook Pro/i)).toBeInTheDocument();
  });

  it("renders manual install message when installed not through Fleet", async () => {
    renderModal({
      details: {
        commandUuid: "",
        fleetInstallStatus: "installed",
        hostDisplayName: "Marko's MacBook Pro",
        appName: "Logic Pro",
      },
    });

    await waitFor(() => {
      expect(screen.getByText(/Logic Pro/i)).toBeInTheDocument();
      expect(screen.getByText(/is installed\./i)).toBeInTheDocument();
    });
  });

  it("renders host label as 'the host' if host name empty", async () => {
    renderModal({
      details: {
        commandUuid: "uuid-installed",
        fleetInstallStatus: "installed",
        hostDisplayName: "",
        appName: "Logic Pro",
      },
    });

    await waitFor(() => {
      expect(screen.getByText(/Fleet installed/i)).toBeInTheDocument();
      expect(screen.getByText(/the host/i)).toBeInTheDocument();
    });
  });

  it("renders Done button by default", async () => {
    const onCancel = jest.fn();

    renderModal({
      onCancel,
      details: {
        commandUuid: "uuid-installed",
        fleetInstallStatus: "installed",
        hostDisplayName: "Marko's MacBook Pro",
        appName: "Logic Pro",
      },
    });

    const doneBtn = await screen.findByRole("button", { name: /done/i });
    doneBtn.click();
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("renders Cancel + Retry when failed_install with deviceAuthToken", async () => {
    const onRetry = jest.fn();
    const onCancel = jest.fn();

    renderModal({
      onRetry,
      onCancel,
      deviceAuthToken: "test_token_123",
      details: {
        commandUuid: "uuid-failed",
        fleetInstallStatus: "failed_install",
        hostDisplayName: "Marko's MacBook Pro",
        appName: "Logic Pro",
      },
      hostSoftware: createMockHostSoftware({
        id: 99,
        name: "CoolApp",
        installed_versions: [],
      }),
    });

    const cancelButton = await screen.findByRole("button", { name: /cancel/i });
    const retryButton = await screen.findByRole("button", { name: /retry/i });

    expect(cancelButton).toBeInTheDocument();
    expect(retryButton).toBeInTheDocument();

    await waitFor(() => {
      retryButton.click();
      expect(onRetry).toHaveBeenCalledWith(99);
      expect(onCancel).toHaveBeenCalled();
    });
  });

  it("renders MDM acknowledged result in details output", async () => {
    renderModal({
      details: {
        commandUuid: "acknowledged-uuid",
        fleetInstallStatus: "pending_install",
        hostDisplayName: "Marko's MacBook Pro",
        appName: "Logic Pro",
      },
    });

    await waitFor(() => {
      expect(
        screen.getByText(
          /acknowledged but the installation has not been verified/i
        )
      ).toBeInTheDocument();
    });
  });
});
