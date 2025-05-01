import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import userEvent from "@testing-library/user-event";
import {
  createMockHostSoftware,
  createMockHostAppStoreApp,
  createMockHostSoftwarePackage,
} from "__mocks__/hostMock";

import InstallStatusCell from "./InstallStatusCell";

// Mock lodash uniqueId to always return the same id for stable tests
jest.mock("lodash", () => ({
  ...jest.requireActual("lodash"),
  uniqueId: jest.fn(() => "test-tooltip-id"),
}));

const baseProps = {
  id: 1,
  name: "Test Software",
  version: "1.0.0",
  source: "test",
  bundle_identifier: "com.test.software",
  hosts_count: 1,
  status: null,
  software_package: null,
  app_store_app: null,
};

const testSoftwarePackage = createMockHostSoftwarePackage();

describe("InstallStatusCell - component", () => {
  it("renders 'Installed' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        {...baseProps}
        status="installed"
        software_package={testSoftwarePackage}
      />
    );

    expect(screen.getByText("Installed")).toBeInTheDocument();

    await user.hover(screen.getByText("Installed"));
    expect(screen.getByText(/Software is installed/i)).toBeInTheDocument();
  });

  it("renders 'Installing (pending)' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        {...baseProps}
        status="pending_install"
        software_package={testSoftwarePackage}
      />
    );

    expect(screen.getByText("Installing (pending)")).toBeInTheDocument();

    await user.hover(screen.getByText("Installing (pending)"));
    expect(
      screen.getByText(/Fleet is installing or will install/i)
    ).toBeInTheDocument();
  });

  it("renders 'Uninstalling (pending)' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        {...baseProps}
        status="pending_uninstall"
        software_package={testSoftwarePackage}
      />
    );

    expect(screen.getByText("Uninstalling (pending)")).toBeInTheDocument();

    await user.hover(screen.getByText("Uninstalling (pending)"));
    expect(
      screen.getByText(/Fleet is uninstalling or will uninstall/i)
    ).toBeInTheDocument();
  });

  it("renders 'Install (failed)' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        {...baseProps}
        status="failed_install"
        software_package={testSoftwarePackage}
      />
    );

    expect(screen.getByText("Install (failed)")).toBeInTheDocument();

    await user.hover(screen.getByText("Install (failed)"));
    expect(
      screen.getByText(/The host failed to install software/i)
    ).toBeInTheDocument();
  });

  it("renders 'Uninstall (failed)' status with tooltip", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        {...baseProps}
        status="failed_uninstall"
        software_package={testSoftwarePackage}
      />
    );

    expect(screen.getByText("Uninstall (failed)")).toBeInTheDocument();

    await user.hover(screen.getByText("Uninstall (failed)"));
    expect(
      screen.getByText(/The host failed to uninstall software/i)
    ).toBeInTheDocument();
  });

  it("renders 'Available for install' for package", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        {...baseProps}
        software_package={testSoftwarePackage}
        status={null}
      />
    );

    expect(screen.getByText("Available for install")).toBeInTheDocument();

    await user.hover(screen.getByText("Available for install"));
    expect(screen.getByText(/can be installed/i)).toBeInTheDocument();
  });

  it("renders 'Available for install' for App Store app", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        {...baseProps}
        software_package={{ ...testSoftwarePackage, self_service: false }}
        status={null}
      />
    );

    expect(screen.getByText("Available for install")).toBeInTheDocument();

    await user.hover(screen.getByText("Available for install"));
    expect(screen.getByText(/can be installed/i)).toBeInTheDocument();
  });

  it("renders 'Self-service' for package with self_service true", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        {...baseProps}
        software_package={{
          ...testSoftwarePackage,
          name: "SelfService Software",
          self_service: true,
        }}
        status={null}
      />
    );

    expect(screen.getAllByText("Self-service").length).toBeGreaterThan(0);

    await user.hover(screen.getAllByText("Self-service")[0]);
    expect(screen.getByText(/can be installed/i)).toBeInTheDocument();
  });

  it("renders 'Self-service' for App Store app with self_service true", async () => {
    const { user } = renderWithSetup(
      <InstallStatusCell
        {...baseProps}
        app_store_app={createMockHostAppStoreApp({ self_service: true })}
        status={null}
      />
    );

    expect(screen.getAllByText("Self-service").length).toBeGreaterThan(0);

    await user.hover(screen.getAllByText("Self-service")[0]);
    expect(screen.getByText(/Software can be installed/i)).toBeInTheDocument();
  });

  it("renders placeholder for missing status and packages", () => {
    render(<InstallStatusCell {...baseProps} />);

    expect(screen.getByText("---")).toBeInTheDocument();
  });
});
