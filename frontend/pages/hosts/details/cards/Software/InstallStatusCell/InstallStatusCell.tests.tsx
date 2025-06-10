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

const testSoftware = createMockHostSoftware();
const testSoftwarePackage = createMockHostSoftwarePackage();

describe("InstallStatusCell - component", () => {
  it("renders 'Installed' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={createMockHostSoftware({
          status: "installed",
          software_package: testSoftwarePackage,
        })}
        onShowSSInstallDetails={noop}
        onShowSSUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Installed")).toBeInTheDocument();

    await user.hover(screen.getByText("Installed"));
  });

  it("renders 'Installing...' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={createMockHostSoftware({
          status: "pending_install",
          software_package: testSoftwarePackage,
        })}
        onShowSSInstallDetails={noop}
        onShowSSUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Installing...")).toBeInTheDocument();

    await user.hover(screen.getByText("Installing..."));
    expect(
      screen.getByText(/Fleet is installing or will install/i)
    ).toBeInTheDocument();
  });

  it("renders 'Uninstalling...' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={createMockHostSoftware({
          status: "pending_uninstall",
          software_package: testSoftwarePackage,
        })}
        onShowSSInstallDetails={noop}
        onShowSSUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Uninstalling...")).toBeInTheDocument();

    await user.hover(screen.getByText("Uninstalling..."));
    expect(
      screen.getByText(/Fleet is uninstalling or will uninstall/i)
    ).toBeInTheDocument();
  });

  it("renders 'Failed' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={createMockHostSoftware({
          status: "failed_install",
          software_package: testSoftwarePackage,
        })}
        onShowSSInstallDetails={noop}
        onShowSSUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Failed")).toBeInTheDocument();

    await user.hover(screen.getByText("Failed"));
    expect(screen.getByText(/Software failed to install/i)).toBeInTheDocument();
  });

  it("renders 'Failed (uninstall)' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={createMockHostSoftware({
          status: "failed_uninstall",
          software_package: testSoftwarePackage,
        })}
        onShowSSInstallDetails={noop}
        onShowSSUninstallDetails={noop}
      />
    );

    expect(screen.getByText("Failed (uninstall)")).toBeInTheDocument();

    await user.hover(screen.getByText("Failed (uninstall)"));
    expect(
      screen.getByText(/Software failed to uninstall/i)
    ).toBeInTheDocument();
  });

  it("renders '---' for package available for install", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={createMockHostSoftware({
          status: null,
          software_package: testSoftwarePackage,
        })}
        onShowSSInstallDetails={noop}
        onShowSSUninstallDetails={noop}
      />
    );

    expect(screen.getByText("---")).toBeInTheDocument();

    await user.hover(screen.getByText("---"));
    expect(screen.getByText(/can be installed/i)).toBeInTheDocument();
  });

  it("renders '---' for App Store app that's available for install", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={createMockHostSoftware({
          status: null,
          software_package: { ...testSoftwarePackage, self_service: false },
        })}
        onShowSSInstallDetails={noop}
        onShowSSUninstallDetails={noop}
      />
    );

    expect(screen.getByText("---")).toBeInTheDocument();

    await user.hover(screen.getByText("---"));
    expect(screen.getByText(/can be installed/i)).toBeInTheDocument();
  });

  it("renders '---' even for package with self_service true", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={createMockHostSoftware({
          status: null,
          software_package: {
            ...testSoftwarePackage,
            name: "SelfService Software",
            self_service: true,
          },
        })}
        onShowSSInstallDetails={noop}
        onShowSSUninstallDetails={noop}
      />
    );

    expect(screen.getAllByText("---").length).toBeGreaterThan(0);

    await user.hover(screen.getAllByText("---")[0]);
    expect(screen.getByText(/can be installed/i)).toBeInTheDocument();
  });

  it("renders '---' even for App Store app with self_service true", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        software={createMockHostSoftware({
          status: null,
          app_store_app: createMockHostAppStoreApp({ self_service: true }),
        })}
        onShowSSInstallDetails={noop}
        onShowSSUninstallDetails={noop}
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
        software={createMockHostSoftware({
          status: null,
          app_store_app: null,
          software_package: null,
        })}
        onShowSSInstallDetails={noop}
        onShowSSUninstallDetails={noop}
      />
    );

    expect(screen.getByText("---")).toBeInTheDocument();
  });
});
