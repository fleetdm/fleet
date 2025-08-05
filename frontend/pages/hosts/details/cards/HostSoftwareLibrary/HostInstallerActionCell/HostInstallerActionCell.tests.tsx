import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import {
  createMockHostAppStoreApp,
  createMockHostSoftware,
  createMockHostSoftwarePackage,
} from "__mocks__/hostMock";

import { noop } from "lodash";
import {
  getActionButtonState,
  HostInstallerActionCell,
} from "./HostInstallerActionCell";

const mockSoftwarePackage = createMockHostSoftwarePackage();
const mockAppStoreApp = createMockHostAppStoreApp();

describe("getButtonActionState helper function", () => {
  it("disables both buttons and sets tooltips when host scripts are off and not an app store app", () => {
    const result = getActionButtonState({
      hostScriptsEnabled: false,
      status: null,
      appStoreApp: null,
      softwareId: 1,
      softwarePackage: mockSoftwarePackage,
      hostMDMEnrolled: false,
    });

    expect(result).toEqual({
      installDisabled: true,
      installTooltip: "To install, turn on host scripts.",
      uninstallDisabled: true,
      uninstallTooltip: "To uninstall, turn on host scripts.",
    });
  });

  it("disables both buttons when status is pending_install", () => {
    const result = getActionButtonState({
      hostScriptsEnabled: true,
      status: "pending_install",
      appStoreApp: null,
      softwareId: 1,
      softwarePackage: mockSoftwarePackage,
      hostMDMEnrolled: true,
    });

    expect(result.installDisabled).toBe(true);
    expect(result.uninstallDisabled).toBe(true);
  });

  it("disables both buttons when status is pending_uninstall", () => {
    const result = getActionButtonState({
      hostScriptsEnabled: true,
      status: "pending_uninstall",
      appStoreApp: null,
      softwareId: 1,
      softwarePackage: mockSoftwarePackage,
      hostMDMEnrolled: true,
    });

    expect(result.installDisabled).toBe(true);
    expect(result.uninstallDisabled).toBe(true);
  });

  it("disables uninstall button for app store app", () => {
    const result = getActionButtonState({
      hostScriptsEnabled: true,
      status: null,
      appStoreApp: mockAppStoreApp,
      softwareId: 1,
      softwarePackage: null,
      hostMDMEnrolled: true,
    });

    expect(result.uninstallDisabled).toBe(true);
    expect(result.installDisabled).toBe(false);
  });

  it("disables install button and sets tooltip for app store app if not enrolled in MDM", () => {
    const result = getActionButtonState({
      hostScriptsEnabled: true,
      status: null,
      appStoreApp: mockAppStoreApp,
      softwareId: 1,
      softwarePackage: null,
      hostMDMEnrolled: false,
    });

    expect(result.installDisabled).toBe(true);
    expect(result.installTooltip).toBe(
      "To install, turn on MDM for this host."
    );
    expect(result.uninstallDisabled).toBe(true);
  });

  it("returns enabled buttons when all conditions are good", () => {
    const result = getActionButtonState({
      hostScriptsEnabled: true,
      status: null,
      appStoreApp: null,
      softwareId: 1,
      softwarePackage: mockSoftwarePackage,
      hostMDMEnrolled: true,
    });

    expect(result.installDisabled).toBe(false);
    expect(result.uninstallDisabled).toBe(false);
    expect(result.installTooltip).toBeUndefined();
    expect(result.uninstallTooltip).toBeUndefined();
  });
});

describe("HostInstallerActionCell component", () => {
  const baseClass = "test";
  const defaultSoftware = createMockHostSoftware();

  it("renders enabled reinstall and uninstall buttons for an installed software", () => {
    render(
      <HostInstallerActionCell
        software={{ ...defaultSoftware, ui_status: "installed" }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    // Install button
    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(installBtn).toHaveTextContent("Reinstall");
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn.closest("button")).not.toBeDisabled();

    // Uninstall button
    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(uninstallBtn).toHaveTextContent("Uninstall");
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
    expect(uninstallBtn.closest("button")).not.toBeDisabled();
  });

  it("disables install button when host scripts are not enabled", () => {
    render(
      <HostInstallerActionCell
        software={{ ...defaultSoftware, ui_status: "installed" }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled={false}
        hostMDMEnrolled={false}
      />
    );

    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(installBtn.closest("button")).toBeDisabled();
  });

  it("does not render uninstall button for app store app", () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          software_package: null,
          app_store_app: mockAppStoreApp,
          ui_status: "installed",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    expect(
      screen.queryByTestId(`${baseClass}__uninstall-button--test`)
    ).toBeNull();
    expect(screen.queryByTestId("trash-icon")).not.toBeInTheDocument();
  });

  it("does not render uninstall button if no softwarePackage", () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          ui_status: "installed",
          software_package: null,
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    expect(
      screen.queryByTestId(`${baseClass}__uninstall-button--test`)
    ).toBeNull();
    expect(screen.queryByTestId("trash-icon")).not.toBeInTheDocument();
  });

  it("updates button text/icon when status changes to non-pending", () => {
    const { rerender } = render(
      <HostInstallerActionCell
        software={{
          ...createMockHostSoftware({ installed_versions: [] }),
          status: "pending_install",
          ui_status: "pending_install",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    // Should show initial text
    expect(
      screen.getByTestId(`${baseClass}__install-button--test`)
    ).toHaveTextContent("Install");
    expect(screen.getByTestId("install-icon")).toBeInTheDocument();

    expect(
      screen.queryByTestId(`${baseClass}__uninstall-button--test`)
    ).toBeNull();
    expect(screen.queryByTestId("trash-icon")).not.toBeInTheDocument();

    // Change status to installed
    rerender(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "installed",
          ui_status: "installed",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    // Should update text
    expect(
      screen.getByTestId(`${baseClass}__install-button--test`)
    ).toHaveTextContent("Reinstall");
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();

    expect(
      screen.getByTestId(`${baseClass}__uninstall-button--test`)
    ).toHaveTextContent("Uninstall");
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
  });

  it("shows tooltip on disabled install button for MDM enrollment", async () => {
    const { user } = renderWithSetup(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          app_store_app: mockAppStoreApp,
          ui_status: "installed",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled={false}
      />
    );
    const btn = screen.getByTestId(`${baseClass}__install-button--test`);
    await user.hover(btn);
    expect(
      screen.getByText(/To install, turn on MDM for this host/)
    ).toBeInTheDocument();
  });

  it('renders correct retry/reinstall for "failed_install" with installed_versions', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "failed_install",
          ui_status: "failed_install",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );
    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn).toHaveTextContent("Retry");

    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
    expect(uninstallBtn).toHaveTextContent("Uninstall");
  });

  it('renders retry and no uninstall for "failed_install" with no installed_versions', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "failed_install",
          ui_status: "failed_install",
          software_package: null,
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );
    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn).toHaveTextContent("Retry");
    // Uninstall does not exist
    expect(
      screen.queryByTestId(`${baseClass}__uninstall-button--test`)
    ).toBeNull();
    expect(screen.queryByTestId("trash-icon")).toBeNull();
  });

  it('renders correct icons/text for "failed_install_update_available" with installed_versions', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "failed_install",
          ui_status: "failed_install_update_available",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );
    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn).toHaveTextContent("Retry");
    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
    expect(uninstallBtn).toHaveTextContent("Uninstall");
  });

  it('renders Reinstall and Retry uninstall for "failed_uninstall" with installed_versions', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "failed_uninstall",
          ui_status: "failed_uninstall",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );
    // Both reinstall and retry uninstall have the same icon
    const refreshIcons = screen.getAllByTestId("refresh-icon");
    expect(refreshIcons).toHaveLength(2);

    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(installBtn).toHaveTextContent("Reinstall");
    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(uninstallBtn).toHaveTextContent("Retry uninstall");
  });

  it('renders Reinstall with no Uninstall button for "failed_uninstall" with no installed_versions', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "failed_uninstall",
          ui_status: "failed_uninstall",
          software_package: null,
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );
    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn).toHaveTextContent("Reinstall");

    expect(
      screen.queryByTestId(`${baseClass}__uninstall-button--test`)
    ).toBeNull();
  });

  it('renders Reinstall and Retry Uninstall for "failed_uninstall_update_available" with installed_versions', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "failed_uninstall",
          ui_status: "failed_uninstall_update_available",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );
    // Both reinstall and retry uninstall have the same icon
    const refreshIcons = screen.getAllByTestId("refresh-icon");
    expect(refreshIcons).toHaveLength(2);

    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(installBtn).toHaveTextContent("Update");
    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(uninstallBtn).toHaveTextContent("Retry uninstall");
  });

  it('renders Update and Uninstall and disables action buttons for "updating"', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "pending_install",
          ui_status: "updating",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );
    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn).toHaveTextContent("Update");
    expect(installBtn.closest("button")).toBeDisabled();

    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
    expect(uninstallBtn).toHaveTextContent("Uninstall");
    expect(uninstallBtn.closest("button")).toBeDisabled();
  });

  it('renders Update and Uninstall and disables action buttons for "pending_update"', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "pending_install",
          ui_status: "pending_update",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );
    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn).toHaveTextContent("Update");
    expect(installBtn.closest("button")).toBeDisabled();

    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
    expect(uninstallBtn).toHaveTextContent("Uninstall");
    expect(uninstallBtn.closest("button")).toBeDisabled();
  });

  it('renders Update and Uninstall for "update_available" ui_status', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "installed",
          ui_status: "update_available",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(installBtn).toHaveTextContent("Update");
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn.closest("button")).not.toBeDisabled();

    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(uninstallBtn).toHaveTextContent("Uninstall");
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
    expect(uninstallBtn.closest("button")).not.toBeDisabled();
  });

  it('renders Reinstall and Uninstall for "uninstalling" ui_status', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "pending_uninstall",
          ui_status: "uninstalling",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(installBtn).toHaveTextContent("Reinstall");
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn.closest("button")).toBeDisabled();

    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(uninstallBtn).toHaveTextContent("Uninstall");
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
    expect(uninstallBtn.closest("button")).toBeDisabled();
  });

  it('renders Reinstall and Uninstall for "pending_uninstall" ui_status', () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          status: "pending_uninstall",
          ui_status: "pending_uninstall",
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(installBtn).toHaveTextContent("Reinstall");
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn.closest("button")).toBeDisabled();

    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(uninstallBtn).toHaveTextContent("Uninstall");
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
    expect(uninstallBtn.closest("button")).toBeDisabled();
  });
  it("renders Reinstall and Uninstall buttons for tgz package with no installed_versions", () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          source: "tgz_packages",
          ui_status: "installed", // could also use "pending_uninstall" or "failed_uninstall"
          status: "installed",
          software_package: mockSoftwarePackage,
          installed_versions: [], // crucial: no versions, triggers the tgzPackageDetectedAsInstalled case
          app_store_app: null,
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(installBtn).toHaveTextContent("Reinstall");
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn.closest("button")).toBeEnabled();

    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(uninstallBtn).toHaveTextContent("Uninstall");
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
    expect(uninstallBtn.closest("button")).toBeEnabled();
  });
  it("renders disabled Reinstall and disabled Uninstall buttons for tgz package with no installed_versions when uninstall pending", () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          source: "tgz_packages",
          ui_status: "pending_uninstall",
          status: "pending_uninstall",
          software_package: mockSoftwarePackage,
          installed_versions: [], // crucial: no versions, triggers the tgzPackageDetectedAsInstalled case
          app_store_app: null,
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(installBtn).toHaveTextContent("Reinstall");
    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();
    expect(installBtn.closest("button")).toBeDisabled();

    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(uninstallBtn).toHaveTextContent("Uninstall");
    expect(screen.getByTestId("trash-icon")).toBeInTheDocument();
    expect(uninstallBtn.closest("button")).toBeDisabled();
  });
  it("renders Reinstall and Retry uninstall buttons for tgz package with no installed_versions when uninstall failed", () => {
    render(
      <HostInstallerActionCell
        software={{
          ...defaultSoftware,
          source: "tgz_packages",
          ui_status: "failed_uninstall",
          status: "failed_uninstall",
          software_package: mockSoftwarePackage,
          installed_versions: [], // crucial: no versions, triggers the tgzPackageDetectedAsInstalled case
          app_store_app: null,
        }}
        onClickInstallAction={noop}
        onClickUninstallAction={noop}
        baseClass={baseClass}
        hostScriptsEnabled
        hostMDMEnrolled
      />
    );

    const installBtn = screen.getByTestId(`${baseClass}__install-button--test`);
    expect(installBtn).toHaveTextContent("Reinstall");
    expect(installBtn.closest("button")).toBeEnabled();

    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(uninstallBtn).toHaveTextContent("Retry uninstall");
    expect(uninstallBtn.closest("button")).toBeEnabled();

    // Both reinstall and retry uninstall have the same icon
    const refreshIcons = screen.getAllByTestId("refresh-icon");
    expect(refreshIcons).toHaveLength(2);
  });
});
