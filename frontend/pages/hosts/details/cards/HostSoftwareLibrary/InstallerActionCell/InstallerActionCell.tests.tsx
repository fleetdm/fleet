import React from "react";
import { render, screen } from "@testing-library/react";
import {
  createMockHostAppStoreApp,
  createMockHostSoftware,
  createMockHostSoftwarePackage,
} from "__mocks__/hostMock";

import { noop } from "lodash";
import {
  getActionButtonState,
  InstallerActionCell,
} from "./InstallerActionCell";

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

describe("InstallerActionCell component", () => {
  const baseClass = "test";
  const defaultSoftware = createMockHostSoftware();

  it("renders install and uninstall buttons with correct text and enabled state", () => {
    render(
      <InstallerActionCell
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
    expect(installBtn.closest("button")).not.toBeDisabled();

    // Uninstall button
    const uninstallBtn = screen.getByTestId(
      `${baseClass}__uninstall-button--test`
    );
    expect(uninstallBtn).toHaveTextContent("Uninstall");
    expect(uninstallBtn.closest("button")).not.toBeDisabled();
  });

  it("disables install button and shows tooltip", () => {
    render(
      <InstallerActionCell
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
      <InstallerActionCell
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
  });

  it("does not render uninstall button if no softwarePackage", () => {
    render(
      <InstallerActionCell
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
  });

  it("updates button text/icon when status changes to non-pending", () => {
    const { rerender } = render(
      <InstallerActionCell
        software={{
          ...defaultSoftware,
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

    // Change status to installed
    rerender(
      <InstallerActionCell
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
  });
});
