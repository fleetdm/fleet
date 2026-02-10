import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup, createCustomRenderer } from "test/test-utils";
import { createMockHostSoftware } from "__mocks__/hostMock";
import { createMockSoftwareInstallResult } from "__mocks__/softwareMock";
import {
  getDefaultSoftwareInstallHandler,
  getSoftwareInstallHandlerNoOutputs,
  getSoftwareInstallHandlerOnlyInstallOutput,
  getSoftwareInstallHandlerWithPreInstall,
  getSoftwareInstallHandlerOnlyPreInstallOutput,
} from "test/handlers/software-handlers";
import mockServer from "test/mock-server";
import { noop } from "lodash";

import SoftwareInstallDetailsModal, {
  StatusMessage,
  ModalButtons,
} from "./SoftwareInstallDetailsModal";

describe("SoftwareInstallDetailsModal", () => {
  describe("StatusMessage component", () => {
    it("renders basic 'is installed' message when not installed by fleet (no installResult provided)", () => {
      render(<StatusMessage softwareName="CoolApp" isMyDevicePage={false} />);
      expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
      expect(screen.getByText(/is installed/)).toBeInTheDocument();
    });

    it("on software library page/pending activity, renders pending install message with host and package name", () => {
      render(
        <StatusMessage
          softwareName="CoolApp"
          installResult={createMockSoftwareInstallResult({
            status: "pending_install",
          })}
          isMyDevicePage={false}
        />
      );

      expect(screen.queryByTestId("pending-outline-icon")).toBeInTheDocument();
      expect(
        screen.getByText(/is installing or will install/)
      ).toBeInTheDocument();
      expect(screen.getByText(/\(com\.cool\.app\)/)).toBeInTheDocument();
      expect(screen.getByText(/Test Host/)).toBeInTheDocument();
      expect(screen.getByText(/when it comes online/)).toBeInTheDocument();
      expect(screen.queryByText(/\d+.*ago/)).not.toBeInTheDocument();
    });

    it("on device user page, renders failed install with retry option with contact link", () => {
      render(
        <StatusMessage
          softwareName="CoolApp"
          installResult={createMockSoftwareInstallResult({
            status: "failed_install",
          })}
          isMyDevicePage
          contactUrl="http://support"
        />
      );

      expect(screen.queryByTestId("error-icon")).toBeInTheDocument();
      expect(screen.getByText(/failed to install/)).toBeInTheDocument();
      expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
      // Host name should not be rendered for device user page
      expect(screen.queryByText(/Test Host/)).not.toBeInTheDocument();
      expect(screen.getByText(/\d+.*ago/)).toBeInTheDocument();
      expect(screen.getByText(/You can retry/)).toBeInTheDocument();
      expect(
        screen.getByRole("link", { name: /contact your IT admin/ })
      ).toHaveAttribute("href", "http://support");
    });

    it("on device user page, renders failed install with retry option without contact link", () => {
      render(
        <StatusMessage
          softwareName="CoolApp"
          installResult={createMockSoftwareInstallResult({
            status: "failed_install",
          })}
          isMyDevicePage
        />
      );

      expect(screen.queryByTestId("error-icon")).toBeInTheDocument();
      expect(screen.getByText(/failed to install/)).toBeInTheDocument();
      expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
      // Host name should not be rendered for device user page
      expect(screen.queryByText(/Test Host/)).not.toBeInTheDocument();
      expect(screen.getByText(/\d+.*ago/)).toBeInTheDocument();
      expect(screen.getByText(/You can retry/)).toBeInTheDocument();
      // Don't show link of not provided
      expect(
        screen.queryByRole("link", { name: /contact your IT admin/ })
      ).not.toBeInTheDocument();
    });

    it("treats failed_install as installed when host still reports installed versions", () => {
      render(
        <StatusMessage
          softwareName="CoolApp"
          installResult={createMockSoftwareInstallResult({
            status: "failed_install",
          })}
          isMyDevicePage={false}
          hasInstalledVersions
        />
      );

      expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
      expect(screen.getByText(/is installed\./i)).toBeInTheDocument();
      expect(screen.queryByText(/failed to install/i)).not.toBeInTheDocument();
    });

    it("on host details page, renders failed install without retry", () => {
      render(
        <StatusMessage
          softwareName="CoolApp"
          installResult={createMockSoftwareInstallResult({
            status: "failed_install",
          })}
          isMyDevicePage={false}
          contactUrl="http://support"
        />
      );

      expect(screen.queryByTestId("error-icon")).toBeInTheDocument();
      expect(screen.getByText(/failed to install/)).toBeInTheDocument();
      expect(screen.getByText(/Test Host/)).toBeInTheDocument();
      expect(screen.queryByText(/You can retry/)).not.toBeInTheDocument();
      expect(screen.getByText(/\d+.*ago/)).toBeInTheDocument();
    });

    it("on host details page/install activity, renders installed message with timestamp", () => {
      render(
        <StatusMessage
          softwareName="CoolApp"
          installResult={createMockSoftwareInstallResult({
            status: "installed",
          })}
          isMyDevicePage={false}
        />
      );

      expect(screen.queryByTestId("success-icon")).toBeInTheDocument();
      expect(screen.getByText(/Fleet installed/)).toBeInTheDocument();
      expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
      expect(screen.getByText(/Test Host/)).toBeInTheDocument();
      expect(screen.getByText(/\(com\.cool\.app\)/)).toBeInTheDocument();
      expect(screen.getByText(/\d+.*ago/)).toBeInTheDocument();
    });
  });

  describe("ModalButtons component", () => {
    it("on device user page, shows Retry/Cancel for failed install and triggers handlers", async () => {
      const onCancel = jest.fn();
      const onRetry = jest.fn();

      const { user } = renderWithSetup(
        <ModalButtons
          deviceAuthToken="token123"
          status="failed_install"
          hostSoftwareId={99}
          onCancel={onCancel}
          onRetry={onRetry}
        />
      );
      expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "Cancel" })
      ).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "Retry" }));
      expect(onRetry).toHaveBeenCalledWith(99);
      expect(onCancel).toHaveBeenCalled();

      await user.click(screen.getByRole("button", { name: "Cancel" }));
      expect(onCancel).toHaveBeenCalledTimes(2);
    });

    it("shows Done button for pending install", () => {
      const onCancel = jest.fn();
      render(<ModalButtons status="pending_install" onCancel={onCancel} />);
      expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Retry" })
      ).not.toBeInTheDocument();
    });

    it("on device user page, shows Done button for installed software", () => {
      const onCancel = jest.fn();
      render(
        <ModalButtons
          deviceAuthToken="token123"
          status="installed"
          onCancel={onCancel}
        />
      );
      expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
    });
  });

  const baseDetails = {
    install_uuid: "uuid-123",
    host_display_name: "Test Host",
  };

  const baseHostSoftware = createMockHostSoftware({
    id: 99,
    name: "CoolApp",
    installed_versions: [],
  });

  describe("Install Details Section", () => {
    afterEach(() => {
      mockServer.resetHandlers();
    });

    it("does not show install details outputs until Details is clicked", async () => {
      mockServer.use(getDefaultSoftwareInstallHandler);
      const renderWithServer = createCustomRenderer({ withBackendMock: true });

      renderWithServer(
        <SoftwareInstallDetailsModal
          details={baseDetails}
          hostSoftware={baseHostSoftware}
          onCancel={noop}
        />
      );

      // waiting for the button to render
      const detailsButton = await screen.findByRole("button", {
        name: /Details/i,
      });

      expect(detailsButton).toBeInTheDocument();
      expect(
        screen.queryByText("Pre-install query output:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText("Install script output:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText(/Post-install script output:/i)
      ).not.toBeInTheDocument();
    });

    it("shows pre-install, install, and post-install outputs after clicking Details", async () => {
      mockServer.use(getSoftwareInstallHandlerWithPreInstall);
      const renderWithServer = createCustomRenderer({ withBackendMock: true });
      const { user } = renderWithServer(
        <SoftwareInstallDetailsModal
          details={baseDetails}
          hostSoftware={baseHostSoftware}
          onCancel={noop}
        />
      );

      const detailsBtn = await screen.findByRole("button", {
        name: /Details/i,
      });
      await user.click(detailsBtn);

      // Pre-install output
      expect(
        await screen.getByText("Pre-install query output:")
      ).toBeInTheDocument();
      expect(screen.getByText("Pre-install check passed")).toBeInTheDocument();

      // Install output
      expect(screen.getByText("Install script output:")).toBeInTheDocument();
      expect(screen.getByText("Install script ran")).toBeInTheDocument();

      // Post-install output
      expect(
        screen.getByText("Post-install script output:")
      ).toBeInTheDocument();
      expect(screen.getByText("Post-install success")).toBeInTheDocument();
    });

    it("renders only pre-install output if that's the only script output present", async () => {
      mockServer.use(getSoftwareInstallHandlerOnlyPreInstallOutput);
      const renderWithServer = createCustomRenderer({ withBackendMock: true });
      const { user } = renderWithServer(
        <SoftwareInstallDetailsModal
          details={baseDetails}
          hostSoftware={baseHostSoftware}
          onCancel={noop}
        />
      );

      const detailsBtn = await screen.findByRole("button", {
        name: /Details/i,
      });
      await user.click(detailsBtn);

      expect(
        await screen.getByText("Pre-install query output:")
      ).toBeInTheDocument();
      expect(screen.getByText(/pre-install only/i)).toBeInTheDocument();
      expect(
        screen.queryByText("Install script output:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText(/Post-install script output:/i)
      ).not.toBeInTheDocument();
    });

    it("shows install and post-install outputs after clicking Details (no pre-install)", async () => {
      mockServer.use(getDefaultSoftwareInstallHandler);
      const renderWithServer = createCustomRenderer({ withBackendMock: true });
      const { user } = renderWithServer(
        <SoftwareInstallDetailsModal
          details={baseDetails}
          hostSoftware={baseHostSoftware}
          onCancel={noop}
        />
      );

      const detailsBtn = await screen.findByRole("button", {
        name: /Details/i,
      });
      await user.click(detailsBtn);

      expect(
        await screen.getByText("Install script output:")
      ).toBeInTheDocument();
      expect(screen.getByText("Install script ran")).toBeInTheDocument();
      expect(
        screen.getByText(/Post-install script output:/i)
      ).toBeInTheDocument();
      expect(screen.getByText("Post-install success")).toBeInTheDocument();
      expect(
        screen.queryByText("Pre-install query output:")
      ).not.toBeInTheDocument();
    });

    it("shows only the install output if post-install and pre-install output is empty", async () => {
      mockServer.use(getSoftwareInstallHandlerOnlyInstallOutput);
      const renderWithServer = createCustomRenderer({ withBackendMock: true });
      const { user } = renderWithServer(
        <SoftwareInstallDetailsModal
          details={baseDetails}
          hostSoftware={baseHostSoftware}
          onCancel={noop}
        />
      );

      const detailsBtn = await screen.findByRole("button", {
        name: /Details/i,
      });
      await user.click(detailsBtn);

      expect(
        await screen.getByText("Install script output:")
      ).toBeInTheDocument();
      expect(screen.getByText(/install only/i)).toBeInTheDocument();
      expect(
        screen.queryByText("Pre-install query output:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText(/Post-install script output:/i)
      ).not.toBeInTheDocument();
    });

    it("does not render details button if all script outputs are empty", async () => {
      mockServer.use(getSoftwareInstallHandlerNoOutputs);
      const renderWithServer = createCustomRenderer({ withBackendMock: true });
      renderWithServer(
        <SoftwareInstallDetailsModal
          details={baseDetails}
          hostSoftware={baseHostSoftware}
          onCancel={noop}
        />
      );

      expect(
        screen.queryByRole("button", {
          name: /Details/i,
        })
      ).not.toBeInTheDocument();
    });
  });
});
