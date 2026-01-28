import React from "react";

import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import { createMockHostSoftware } from "__mocks__/hostMock";
import { IHostSoftwareUiStatus } from "interfaces/software";

import { StatusMessage, ModalButtons } from "./SoftwareUninstallDetailsModal";

describe("SoftwareUninstallDetailsModal - StatusMessage component", () => {
  it("from activity or host details page of offline host, renders pending uninstall message with package name and host", () => {
    render(
      <StatusMessage
        hostDisplayName="Offline Host"
        status="pending_uninstall"
        softwareName="CoolApp"
        softwarePackageName="com.cool.app"
        isMyDevicePage={false}
      />
    );

    expect(screen.queryByTestId("pending-outline-icon")).toBeInTheDocument();
    expect(
      screen.getByText(/is uninstalling or will uninstall/)
    ).toBeInTheDocument();
    expect(screen.getByText(/\(com\.cool\.app\)/)).toBeInTheDocument(); // Show package name
    expect(screen.getByText(/Offline Host/)).toBeInTheDocument(); // Show host name
    expect(screen.getByText(/when it comes online/)).toBeInTheDocument(); // Only reach modal if host offline
  });

  // from device user page, cannot reach pending uninstall modal

  it("from device user page, renders failed uninstall message with retry text", () => {
    render(
      <StatusMessage
        hostDisplayName="Test Host"
        status="failed_uninstall"
        softwareName="CoolApp"
        isMyDevicePage
        contactUrl="http://support"
      />
    );

    expect(screen.queryByTestId("error-icon")).toBeInTheDocument();
    expect(screen.getByText(/failed to uninstall/)).toBeInTheDocument();
    expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
    expect(screen.queryByText(/Test Host/)).not.toBeInTheDocument(); // Do not render host name
    expect(screen.getByText(/You can retry/)).toBeInTheDocument(); // Render retry message
  });

  it("from host details page/failed uninstall activity, renders failed uninstall message for with no retry text", () => {
    render(
      <StatusMessage
        hostDisplayName="Test Host"
        status="failed_uninstall"
        softwareName="CoolApp"
        isMyDevicePage={false}
        contactUrl="http://support"
      />
    );

    expect(screen.queryByTestId("error-icon")).toBeInTheDocument();
    expect(screen.getByText(/failed to uninstall/)).toBeInTheDocument();
    expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
    expect(screen.getByText(/Test Host/)).toBeInTheDocument(); // Render host name
    expect(screen.queryByText(/You can retry/)).not.toBeInTheDocument(); // Do not render retry message
  });

  it("from successful uninstall activity, renders uninstalled message with timestamp", () => {
    render(
      <StatusMessage
        hostDisplayName="Test Host"
        status="uninstalled"
        softwareName="CoolApp"
        softwarePackageName="com.cool.app"
        timestamp="2025-08-10T12:00:00Z"
        isMyDevicePage={false}
      />
    );

    expect(screen.queryByTestId("success-icon")).toBeInTheDocument();
    expect(screen.getByText(/Fleet uninstalled/)).toBeInTheDocument();
    expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
    expect(screen.getByText(/Test Host/)).toBeInTheDocument();
    expect(screen.getByText(/\(com\.cool\.app\)/)).toBeInTheDocument();
    expect(screen.getByText(/\d+.*ago/)).toBeInTheDocument(); // timestamp relative
  });

  // from device user page, cannot reach successful uninstall modal
});

describe("SoftwareUninstallDetailsModal - ModalButtons component", () => {
  it("from failed uninstall on a device user page, shows Retry/Cancel and calls handlers", async () => {
    const onCancel = jest.fn();
    const onRetry = jest.fn();
    const hostSoftware = {
      ...createMockHostSoftware({ status: "failed_uninstall" }),
      ui_status: "failed_uninstall" as IHostSoftwareUiStatus,
    };

    const { user } = renderWithSetup(
      <ModalButtons
        uninstallStatus="failed_uninstall"
        deviceAuthToken="token123"
        onCancel={onCancel}
        onRetry={onRetry}
        hostSoftware={hostSoftware}
      />
    );

    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledWith(hostSoftware);
    expect(onCancel).toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onCancel).toHaveBeenCalledTimes(2); // first from retry, second from cancel
  });

  it("from pending uninstall activity or software library of an offline host, shows only Done button", () => {
    const onCancel = jest.fn();

    render(
      <ModalButtons uninstallStatus="pending_uninstall" onCancel={onCancel} />
    );

    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Retry" })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Cancel" })
    ).not.toBeInTheDocument();
  });

  // from device user page, cannot reach pending uninstall modal

  it("from successful uninstall activity, shows Done", () => {
    const onCancel = jest.fn();

    render(
      <ModalButtons
        uninstallStatus="uninstalled"
        deviceAuthToken="token123"
        onCancel={onCancel}
      />
    );

    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
  });

  // from device user page, cannot reach successful uninstall modal
});
