import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import { createMockSoftwareInstallResult } from "__mocks__/softwareMock";
import { ISoftwareScriptResult } from "interfaces/software";
import { StatusMessage, ModalButtons } from "./SoftwareScriptDetailsModal";

describe("SoftwareScriptDetailsModal - StatusMessage component", () => {
  it("on software library page/pending activity, renders pending install message with host and package name", () => {
    render(
      <StatusMessage
        installResult={
          createMockSoftwareInstallResult({
            status: "pending_install",
          }) as ISoftwareScriptResult
        }
        isMyDevicePage={false}
      />
    );

    expect(screen.queryByTestId("pending-outline-icon")).toBeInTheDocument();
    expect(screen.getByText(/is running or will run/)).toBeInTheDocument();
    expect(screen.getByText(/\(com\.cool\.app\)/)).toBeInTheDocument();
    expect(screen.getByText(/Test Host/)).toBeInTheDocument();
    expect(screen.getByText(/when it comes online/)).toBeInTheDocument();
  });

  it("on device user page, renders failed run with rerun option with contact link", () => {
    render(
      <StatusMessage
        installResult={
          createMockSoftwareInstallResult({
            status: "failed_install",
          }) as ISoftwareScriptResult
        }
        isMyDevicePage
        contactUrl="http://support"
      />
    );

    expect(screen.queryByTestId("error-icon")).toBeInTheDocument();
    expect(screen.getByText(/failed to run/)).toBeInTheDocument();
    expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
    // Host name should not be rendered for device user page
    expect(screen.queryByText(/Test Host/)).not.toBeInTheDocument();
    expect(screen.getByText(/You can rerun/)).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /contact your IT admin/ })
    ).toHaveAttribute("href", "http://support");
  });

  it("on device user page, renders failed install with retry option without contact link", () => {
    render(
      <StatusMessage
        installResult={
          createMockSoftwareInstallResult({
            status: "failed_install",
          }) as ISoftwareScriptResult
        }
        isMyDevicePage
      />
    );

    expect(screen.queryByTestId("error-icon")).toBeInTheDocument();
    expect(screen.getByText(/failed to run/)).toBeInTheDocument();
    expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
    // Host name should not be rendered for device user page
    expect(screen.queryByText(/Test Host/)).not.toBeInTheDocument();
    expect(screen.getByText(/You can rerun/)).toBeInTheDocument();
    // Don't show link of not provided
    expect(
      screen.queryByRole("link", { name: /contact your IT admin/ })
    ).not.toBeInTheDocument();
  });

  it("on host details page, renders failed script without rerun", () => {
    render(
      <StatusMessage
        installResult={
          createMockSoftwareInstallResult({
            status: "failed_install",
          }) as ISoftwareScriptResult
        }
        isMyDevicePage={false}
        contactUrl="http://support"
      />
    );

    expect(screen.queryByTestId("error-icon")).toBeInTheDocument();
    expect(screen.getByText(/failed to run/)).toBeInTheDocument();
    expect(screen.getByText(/Test Host/)).toBeInTheDocument();
    expect(screen.queryByText(/You can rerun/)).not.toBeInTheDocument();
  });

  it("on host details page/install activity, renders ran message with timestamp", () => {
    render(
      <StatusMessage
        installResult={
          createMockSoftwareInstallResult({
            status: "installed",
          }) as ISoftwareScriptResult
        }
        isMyDevicePage={false}
      />
    );

    expect(screen.queryByTestId("success-icon")).toBeInTheDocument();
    expect(screen.getByText(/Fleet ran/)).toBeInTheDocument();
    expect(screen.getByText(/CoolApp/)).toBeInTheDocument();
    expect(screen.getByText(/Test Host/)).toBeInTheDocument();
    expect(screen.getByText(/\(com\.cool\.app\)/)).toBeInTheDocument();
    expect(screen.getByText(/\d+.*ago/)).toBeInTheDocument();
  });
});

describe("SoftwareScriptDetailsModal - ModalButtons component", () => {
  it("on device user page, shows Rerun/Cancel for failed install and triggers handlers", async () => {
    const onCancel = jest.fn();
    const onRerun = jest.fn();

    const { user } = renderWithSetup(
      <ModalButtons
        deviceAuthToken="token123"
        installResultStatus="failed_install"
        hostSoftwareId={99}
        onCancel={onCancel}
        onRerun={onRerun}
      />
    );
    expect(screen.getByRole("button", { name: "Rerun" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Rerun" }));
    expect(onRerun).toHaveBeenCalledWith(99);
    expect(onCancel).toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onCancel).toHaveBeenCalledTimes(2);
  });

  it("shows Done button for pending run", () => {
    const onCancel = jest.fn();
    render(
      <ModalButtons installResultStatus="pending_script" onCancel={onCancel} />
    );
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Rerun" })
    ).not.toBeInTheDocument();
  });

  it("on device user page, shows Done button for ran payload-free software", () => {
    const onCancel = jest.fn();
    render(
      <ModalButtons
        deviceAuthToken="token123"
        installResultStatus="installed"
        onCancel={onCancel}
      />
    );
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
  });
});
