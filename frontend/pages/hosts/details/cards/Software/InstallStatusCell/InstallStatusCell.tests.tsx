import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import {
  createMockHostSoftware,
  createMockHostAppStoreApp,
  createMockHostSoftwarePackage,
} from "__mocks__/hostMock";
import { noop } from "lodash";

import InstallStatusCell from "./InstallStatusCell";

// Mock lodash uniqueId to always return the same id for stable tests
jest.mock("lodash", () => ({
  ...jest.requireActual("lodash"),
  uniqueId: jest.fn(() => "test-tooltip-id"),
}));

const testSoftwarePackage = createMockHostSoftwarePackage();

describe("InstallStatusCell - component", () => {
  it("renders 'Installed' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: "installed",
            software_package: testSoftwarePackage,
          }),
          ui_status: "installed",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Installed")).toBeInTheDocument();
    expect(screen.getByTestId("success-icon")).toBeInTheDocument();

    await user.hover(screen.getByText("Installed"));

    // There SHOULD be a button with this label
    expect(
      screen.queryByRole("button", { name: /installed/i })
    ).toBeInTheDocument();
  });

  it("renders 'Installed' status with tooltip button even if not installed via Fleet", () => {
    renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: "installed",
            software_package: createMockHostSoftwarePackage({
              last_install: null,
            }),
          }),
          ui_status: "installed",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Installed")).toBeInTheDocument();
    expect(screen.getByTestId("success-icon")).toBeInTheDocument();

    // There should NOT be a button with this label
    expect(
      screen.queryByRole("button", { name: /installed/i })
    ).toBeInTheDocument();
  });

  it("renders 'Installing...' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: "pending_install",
            software_package: testSoftwarePackage,
          }),
          ui_status: "installing",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
        isHostOnline
      />
    );

    expect(screen.getByText("Installing...")).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();

    await user.hover(screen.getByText("Installing..."));
    expect(
      screen.getByText(/Fleet is installing software./i)
    ).toBeInTheDocument();
  });

  it("renders 'Install (pending)' status with tooltip if host is offline", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: "pending_install",
            software_package: testSoftwarePackage,
          }),
          ui_status: "pending_install",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Install (pending)")).toBeInTheDocument();
    expect(screen.getByTestId("pending-outline-icon")).toBeInTheDocument();

    await user.hover(screen.getByText("Install (pending)"));
    expect(
      screen.getByText(/Fleet will install software/i)
    ).toBeInTheDocument();
  });

  it("renders 'Uninstalling...' status with tooltip if host is online", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: "pending_uninstall",
            software_package: testSoftwarePackage,
          }),
          ui_status: "uninstalling",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
        isHostOnline
      />
    );

    expect(screen.getByText("Uninstalling...")).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();

    await user.hover(screen.getByText("Uninstalling..."));
    expect(
      screen.getByText(/Fleet is uninstalling software./i)
    ).toBeInTheDocument();
  });

  it("renders 'Uninstall (pending)' status with tooltip if host is offline", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: "pending_uninstall",
            software_package: testSoftwarePackage,
          }),
          ui_status: "pending_uninstall",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Uninstall (pending)")).toBeInTheDocument();
    expect(screen.getByTestId("pending-outline-icon")).toBeInTheDocument();

    await user.hover(screen.getByText("Uninstall (pending)"));
    expect(
      screen.getByText(/Fleet will uninstall software/i)
    ).toBeInTheDocument();
  });

  it("renders 'Failed' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: "failed_install",
            software_package: testSoftwarePackage,
          }),
          ui_status: "failed_install",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Failed")).toBeInTheDocument();
    expect(screen.getByTestId("error-icon")).toBeInTheDocument();

    await user.hover(screen.getByText("Failed"));
    expect(screen.getByText(/Software failed to install/i)).toBeInTheDocument();
  });

  it("renders 'Failed (uninstall)' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: "failed_uninstall",
            software_package: testSoftwarePackage,
          }),
          ui_status: "failed_uninstall",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Failed (uninstall)")).toBeInTheDocument();
    expect(screen.getByTestId("error-icon")).toBeInTheDocument();

    await user.hover(screen.getByText("Failed (uninstall)"));
    expect(
      screen.getByText(/Software failed to uninstall/i)
    ).toBeInTheDocument();
  });

  it("renders 'Update available' for failed_install_update_available", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: "failed_install",
            software_package: testSoftwarePackage,
          }),
          ui_status: "failed_install_update_available",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );
    expect(screen.getByText("Update available")).toBeInTheDocument();
    expect(screen.getByTestId("error-outline-icon")).toBeInTheDocument();
    await user.hover(screen.getByText("Update available"));
    expect(screen.getByText(/Fleet can update software/i)).toBeInTheDocument();
  });

  it("renders 'Update available' for failed_uninstall_update_available", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: "failed_uninstall",
            software_package: testSoftwarePackage,
          }),
          ui_status: "failed_uninstall_update_available",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );
    expect(screen.getByText("Update available")).toBeInTheDocument();
    expect(screen.getByTestId("error-outline-icon")).toBeInTheDocument();
    await user.hover(screen.getByText("Update available"));
    expect(screen.getByText(/Fleet can update software/i)).toBeInTheDocument();
  });

  it("renders 'Update available' for status null but update_available", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: null,
            software_package: testSoftwarePackage,
          }),
          ui_status: "update_available",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );
    expect(screen.getByText("Update available")).toBeInTheDocument();
    expect(screen.getByTestId("error-outline-icon")).toBeInTheDocument();
    await user.hover(screen.getByText("Update available"));
    expect(screen.getByText(/Fleet can update software/i)).toBeInTheDocument();
  });

  it("renders '---' for package available for install", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: null,
            software_package: testSoftwarePackage,
          }),
          ui_status: "uninstalled",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getByText("---")).toBeInTheDocument();

    await user.hover(screen.getByText("---"));
    expect(screen.getByText(/can be installed/i)).toBeInTheDocument();
  });

  it("renders '---' for App Store app that's available for install", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: null,
            software_package: { ...testSoftwarePackage, self_service: false },
          }),
          ui_status: "uninstalled",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getByText("---")).toBeInTheDocument();

    await user.hover(screen.getByText("---"));
    expect(screen.getByText(/can be installed/i)).toBeInTheDocument();
  });

  it("renders '---' even for package with self_service true", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: null,
            software_package: {
              ...testSoftwarePackage,
              name: "SelfService Software",
              self_service: true,
            },
          }),
          ui_status: "uninstalled",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getAllByText("---").length).toBeGreaterThan(0);

    await user.hover(screen.getAllByText("---")[0]);
    expect(screen.getByText(/can be installed/i)).toBeInTheDocument();
  });

  it("renders '---' even for App Store app with self_service true", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: null,
            app_store_app: createMockHostAppStoreApp({ self_service: true }),
          }),
          ui_status: "uninstalled",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getAllByText("---").length).toBeGreaterThan(0);

    await user.hover(screen.getAllByText("---")[0]);
    expect(
      screen.getByText(/App store app can be installed/i)
    ).toBeInTheDocument();
  });

  it("renders placeholder for missing status and packages", () => {
    render(
      <InstallStatusCell
        software={{
          ...createMockHostSoftware({
            status: null,
            app_store_app: null,
            software_package: null,
          }),
          ui_status: "uninstalled",
        }}
        onShowInstallDetails={noop}
        onShowUninstallDetails={noop}
      />
    );

    expect(screen.getByText("---")).toBeInTheDocument();
  });
});
