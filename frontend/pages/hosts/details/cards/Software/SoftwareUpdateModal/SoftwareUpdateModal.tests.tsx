import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import { noop } from "lodash";
import { createMockHostSoftware } from "__mocks__/hostMock";
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
      screen.queryByRole("button", { name: "Done" })
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

  it("shows 'Done' button and not update/cancel when status is pending_install", () => {
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
    expect(screen.queryByRole("button", { name: "Done" })).toBeInTheDocument();
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

    expect(screen.getByText(/New version of/i)).toBeInTheDocument();
    expect(screen.getByText(/mock software.app/i)).toBeInTheDocument();
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
});
