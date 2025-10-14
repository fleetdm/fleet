import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import {
  getStatusMessage,
  ModalButtons,
} from "./SoftwareIpaInstallDetailsModal";

describe("getStatusMessage helper function", () => {
  it("shows NotNow message when isStatusNotNow is true", () => {
    render(
      getStatusMessage({
        displayStatus: "pending_install",
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

  it("shows failed_install message", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
      })
    );
    expect(
      screen.getByText(/Please re-attempt this installation/i)
    ).toBeInTheDocument();
  });

  it("shows failed verification message", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
        commandUpdatedAt: "2025-07-29T22:49:52Z",
      })
    );
    expect(
      screen.getByText(
        /but the installation has not been verified. Please re-attempt this installation/i
      )
    ).toBeInTheDocument();
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
  it("renders Done button by default", async () => {
    const onCancel = jest.fn();

    const { user } = renderWithSetup(
      <ModalButtons displayStatus="installed" onCancel={onCancel} />
    );

    const doneButton = screen.getByRole("button", { name: /done/i });
    expect(doneButton).toBeInTheDocument();

    await user.click(doneButton);
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
