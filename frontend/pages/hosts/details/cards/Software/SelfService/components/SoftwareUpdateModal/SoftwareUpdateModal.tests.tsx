import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import {
  createMockHostSoftware,
  createMockHostSoftwarePackage,
} from "__mocks__/hostMock";
import { renderWithSetup } from "test/test-utils";

import SoftwareUpdateModal from "./SoftwareUpdateModal";

describe("SoftwareUpdateModal", () => {
  it("shows modal title and both buttons in update scenario", async () => {
    const mockSoftware = createMockHostSoftware();

    const onExit = jest.fn();
    const onUpdate = jest.fn();

    const { user } = renderWithSetup(
      <SoftwareUpdateModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        onExit={onExit}
        onUpdate={onUpdate}
      />
    );

    expect(screen.getByText("Update details")).toBeVisible();

    expect(
      screen.queryByRole("button", { name: "Close" })
    ).not.toBeInTheDocument();

    // Click Update: both handlers called
    await user.click(screen.getByRole("button", { name: "Update" }));
    expect(onUpdate).toHaveBeenCalledWith(mockSoftware.id);
    expect(onExit).toHaveBeenCalledTimes(1);

    // Click Cancel: only exit handler called
    onExit.mockClear();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onExit).toHaveBeenCalledTimes(1);
    expect(onUpdate).toHaveBeenCalledTimes(1); // shouldn't increment from previous
  });

  it("shows 'Close' button and hides Update/Cancel when disableUpdate is true", () => {
    const mockSoftware = createMockHostSoftware();

    render(
      <SoftwareUpdateModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        onExit={noop}
        onUpdate={noop}
        disableUpdate
      />
    );
    expect(
      screen.queryByRole("button", { name: "Update" })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Cancel" })
    ).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
  });

  it("shows 'Close' button and not update/cancel when status is pending_install", () => {
    const mockSoftware = createMockHostSoftware({ status: "pending_install" });

    render(
      <SoftwareUpdateModal
        hostDisplayName="Offline Workstation"
        software={mockSoftware}
        onExit={noop}
        onUpdate={noop}
      />
    );
    expect(
      screen.queryByRole("button", { name: "Cancel" })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Update" })
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Close" })).toBeInTheDocument();
  });

  it("renders correct status message for 'pending_install'", () => {
    const mockSoftware = createMockHostSoftware({ status: "pending_install" });
    render(
      <SoftwareUpdateModal
        hostDisplayName="Offline Laptop"
        software={mockSoftware}
        onExit={noop}
        onUpdate={noop}
      />
    );

    expect(screen.queryByTestId("pending-outline-icon")).toBeInTheDocument();
    expect(
      screen.getByText(/Fleet is updating or will update/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/Offline Laptop/)).toBeInTheDocument();
    expect(screen.getByText(/\(mock software\.app\)/)).toBeInTheDocument();
  });

  it("renders device user message", () => {
    const mockSoftware = createMockHostSoftware({
      status: "pending_install",
    });
    render(
      <SoftwareUpdateModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        isDeviceUser
        onExit={noop}
        onUpdate={noop}
      />
    );

    expect(screen.queryByTestId("pending-outline-icon")).toBeInTheDocument();
    expect(
      screen.getByText(/Fleet is updating or will update/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/\(mock software\.app\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Test Host/i)).toBeInTheDocument();
    expect(screen.getByText(/when it comes online/i)).toBeInTheDocument();
  });

  it("renders generic message for host details > software > library", () => {
    const mockSoftware = createMockHostSoftware({});
    render(
      <SoftwareUpdateModal
        hostDisplayName="Test PC"
        software={mockSoftware}
        onExit={noop}
        onUpdate={noop}
      />
    );

    expect(screen.queryByTestId("error-outline-icon")).toBeInTheDocument();
    expect(screen.getByText(/New version of/i)).toBeInTheDocument();
    expect(screen.getByText(/Mock Software/i)).toBeInTheDocument();
    expect(
      screen.getByText(/Update the current version on/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/Test PC/i)).toBeInTheDocument();
  });

  it("renders 'Current version' if exactly one version is installed", () => {
    const mockSoftware = createMockHostSoftware({});
    render(
      <SoftwareUpdateModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        onExit={noop}
        onUpdate={noop}
      />
    );
    expect(screen.getByText("Current version:")).toBeInTheDocument();
  });

  it("renders 'Current versions' if more than one installed", () => {
    const mockSoftware = createMockHostSoftware({
      installed_versions: [
        {
          version: "1.0.0",
          bundle_identifier: "abc",
          last_opened_at: "2023-01-01T00:00:00Z",
          vulnerabilities: [],
          installed_paths: ["/Applications/test.appA"],
        },
        {
          version: "2.0.0",
          bundle_identifier: "xyz",
          last_opened_at: "2024-06-01T00:00:00Z",
          vulnerabilities: [],
          installed_paths: ["/Applications/test.appB"],
        },
      ],
    });
    render(
      <SoftwareUpdateModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        onExit={noop}
        onUpdate={noop}
      />
    );
    expect(screen.getByText("Current versions:")).toBeInTheDocument();
  });

  // Shouldn't happen, unless tarballs or weird edge case
  it("does not render current versions if list is empty", () => {
    const mockSoftware = createMockHostSoftware({
      installed_versions: [],
      status: "installed",
    });
    render(
      <SoftwareUpdateModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        onExit={noop}
        onUpdate={noop}
      />
    );
    expect(screen.queryByText(/Current version:/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Current versions:/i)).not.toBeInTheDocument();
  });

  it("does not render current versions if status is 'pending_install'", () => {
    const mockSoftware = createMockHostSoftware({
      status: "pending_install",
    });
    render(
      <SoftwareUpdateModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        onExit={noop}
        onUpdate={noop}
      />
    );
    expect(screen.queryByText(/Current version:/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Current versions:/i)).not.toBeInTheDocument();
  });

  describe("older versions at other paths", () => {
    const makeVersion = (version: string, path: string) => ({
      version,
      bundle_identifier: "com.google.Chrome",
      last_opened_at: "2024-06-01T00:00:00Z",
      vulnerabilities: [],
      installed_paths: [path],
    });
    const renderModal = (
      overrides: Parameters<typeof createMockHostSoftware>[0]
    ) =>
      render(
        <SoftwareUpdateModal
          hostDisplayName="Test Host"
          software={createMockHostSoftware({
            software_package: createMockHostSoftwarePackage({
              version: "2.0.0",
            }),
            ...overrides,
          })}
          onExit={noop}
          onUpdate={noop}
          isDeviceUser
        />
      );
    const STALE_COPY = /To clear this update, delete the older/;

    it("explains an outdated copy next to an up-to-date one", () => {
      renderModal({
        installed_versions: [
          makeVersion("2.0.0", "/Applications/Google Chrome.app"),
          makeVersion("1.0.0", "/Applications/Google Chrome 2.app"),
        ],
      });
      expect(screen.getByText(STALE_COPY)).toBeInTheDocument();
      expect(
        screen.getByText(/An older version is installed at/)
      ).toBeVisible();
      // Path is also listed in the versions card
      expect(
        screen.getAllByText("/Applications/Google Chrome 2.app").length
      ).toBeGreaterThan(1);
    });

    it("explains an outdated copy outside the default location", () => {
      renderModal({
        installed_versions: [
          makeVersion("1.0.0", "/Users/test/Downloads/Google Chrome.app"),
        ],
      });
      expect(screen.getByText(STALE_COPY)).toBeInTheDocument();
    });

    it("pluralizes multiple outdated copies", () => {
      renderModal({
        installed_versions: [
          makeVersion("2.0.0", "/Applications/Google Chrome.app"),
          makeVersion("1.0.0", "/Applications/Google Chrome 2.app"),
          makeVersion("1.5.0", "/Users/test/Applications/Google Chrome.app"),
        ],
      });
      expect(
        screen.getByText(/Older versions are installed at/)
      ).toBeInTheDocument();
      expect(
        screen.getByText(/delete the older versions\./)
      ).toBeInTheDocument();
    });

    it("does not explain a single outdated copy at the default location", () => {
      renderModal({
        installed_versions: [
          makeVersion("1.0.0", "/Applications/Google Chrome.app"),
        ],
      });
      expect(screen.queryByText(STALE_COPY)).not.toBeInTheDocument();
    });

    it("does not explain outdated copies for non-macOS software", () => {
      renderModal({
        source: "programs",
        installed_versions: [
          makeVersion("2.0.0", "C:\\Program Files\\Chrome"),
          makeVersion("1.0.0", "C:\\Users\\test\\Chrome"),
        ],
      });
      expect(screen.queryByText(STALE_COPY)).not.toBeInTheDocument();
    });

    it("does not explain outdated copies while the update is pending", () => {
      renderModal({
        status: "pending_install",
        installed_versions: [
          makeVersion("2.0.0", "/Applications/Google Chrome.app"),
          makeVersion("1.0.0", "/Applications/Google Chrome 2.app"),
        ],
      });
      expect(screen.queryByText(STALE_COPY)).not.toBeInTheDocument();
    });
  });
});
