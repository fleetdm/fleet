import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostSoftware } from "__mocks__/hostMock";
import InventoryVersionsModal from "./InventoryVersionsModal";

// Mock current time for time stamp test
beforeAll(() => {
  jest.useFakeTimers();
  jest.setSystemTime(new Date("2022-05-08T10:00:00Z"));
});

describe("SoftwareDetailsModal", () => {
  it("renders details including hash, vulnerabilities, and paths", () => {
    const mockSoftware = createMockHostSoftware();
    render(
      <InventoryVersionsModal hostSoftware={mockSoftware} onExit={jest.fn()} />
    );

    // Modal title
    expect(screen.getByText(mockSoftware.name)).toBeVisible();

    // Version, Type, Bundle identifier, Last opened
    expect(screen.getByText("Version")).toBeVisible();
    expect(screen.getByText("1.0.0")).toBeVisible();
    expect(screen.getByText("Type")).toBeVisible();
    expect(screen.getByText("Application (macOS)")).toBeVisible();
    expect(screen.getByText("Bundle identifier")).toBeVisible();
    expect(screen.getByText("com.test.mock")).toBeVisible();
    expect(screen.getByText("Last opened")).toBeVisible();
    expect(screen.getByText("4 months ago")).toBeVisible();

    // File path
    expect(screen.getByText("Path:")).toBeVisible();
    expect(screen.getByText("/Applications/mock.app")).toBeVisible();

    // Hash
    expect(screen.getByText("Hash:")).toBeVisible();
    expect(screen.getByText("mockhashhere")).toBeVisible();

    // Vulnerabilities
    expect(screen.getByText(/CVE-2020-0001/)).toBeVisible();

    // Done button
    expect(screen.getByRole("button", { name: "Done" })).toBeVisible();
  });

  it("does not render hash if signature_information is missing", () => {
    const mockSoftware = createMockHostSoftware({
      installed_versions: [
        {
          version: "1.0.0",
          last_opened_at: "2022-01-01T12:00:00Z",
          vulnerabilities: ["CVE-2020-0001"],
          installed_paths: ["/Applications/mock.app"],
          bundle_identifier: "com.mock.software",
          signature_information: undefined,
        },
      ],
    });
    render(
      <InventoryVersionsModal hostSoftware={mockSoftware} onExit={jest.fn()} />
    );

    expect(screen.queryByText("Hash:")).not.toBeInTheDocument();
    expect(screen.queryByText("mockhashhere")).not.toBeInTheDocument();
  });

  it("renders only software type if there are no installed versions", () => {
    const mockSoftware = createMockHostSoftware({ installed_versions: [] });
    render(
      <InventoryVersionsModal hostSoftware={mockSoftware} onExit={jest.fn()} />
    );
    expect(screen.getByText("Type")).toBeVisible();
    expect(screen.getByText("Application (macOS)")).toBeVisible();
    expect(screen.queryByText("Version")).not.toBeInTheDocument();
    expect(screen.queryByText("Path:")).not.toBeInTheDocument();
  });

  it("renders multiple file paths and their corresponding hashes", () => {
    const mockSoftware = createMockHostSoftware({
      installed_versions: [
        {
          version: "2.0.0",
          last_opened_at: "2022-02-01T12:00:00Z",
          vulnerabilities: [],
          installed_paths: ["/Applications/foo.app", "/Applications/bar.app"],
          bundle_identifier: "com.example.multi",
          signature_information: [
            {
              installed_path: "/Applications/foo.app",
              team_identifier: "TEAM1",
              hash_sha256: "hashfoo123",
            },
            {
              installed_path: "/Applications/bar.app",
              team_identifier: "TEAM2",
              hash_sha256: "hashbar456",
            },
          ],
        },
      ],
    });

    render(
      <InventoryVersionsModal hostSoftware={mockSoftware} onExit={jest.fn()} />
    );

    // File paths
    expect(screen.getByText("/Applications/foo.app")).toBeVisible();
    expect(screen.getByText("/Applications/bar.app")).toBeVisible();

    // Hashes
    expect(screen.getByText("hashfoo123")).toBeVisible();
    expect(screen.getByText("hashbar456")).toBeVisible();
  });

  it("renders 'Never' for last opened when last_opened_at is null and source is apps", () => {
    const mockSoftware = createMockHostSoftware({
      source: "apps",
      installed_versions: [
        {
          version: "1.0.0",
          last_opened_at: null,
          vulnerabilities: ["CVE-2020-0001"],
          installed_paths: ["/Applications/mock.app"],
          bundle_identifier: "com.mock.software",
          signature_information: [
            {
              installed_path: "/Applications/mock.app",
              team_identifier: "12345TEAMIDENT",
              hash_sha256: "mockhashhere",
            },
          ],
        },
      ],
    });

    render(
      <InventoryVersionsModal hostSoftware={mockSoftware} onExit={jest.fn()} />
    );

    expect(screen.getByText("Last opened")).toBeVisible();
    expect(screen.getByText("Never")).toBeVisible();
  });

  it("renders 'Never' for last opened when last_opened_at is null and source is programs", () => {
    const mockSoftware = createMockHostSoftware({
      source: "programs",
      installed_versions: [
        {
          version: "1.0.0",
          last_opened_at: null,
          vulnerabilities: ["CVE-2020-0001"],
          installed_paths: ["C:\\Program Files\\mock.exe"],
          bundle_identifier: "",
          signature_information: [
            {
              installed_path: "C:\\Program Files\\mock.exe",
              team_identifier: "",
              hash_sha256: "mockhashhere",
            },
          ],
        },
      ],
    });

    render(
      <InventoryVersionsModal hostSoftware={mockSoftware} onExit={jest.fn()} />
    );

    expect(screen.getByText("Last opened")).toBeVisible();
    expect(screen.getByText("Never")).toBeVisible();
  });

  it("does not render last opened field when last_opened_at is null and source is not apps or programs", () => {
    const mockSoftware = createMockHostSoftware({
      source: "deb_packages",
      installed_versions: [
        {
          version: "1.0.0",
          last_opened_at: null,
          vulnerabilities: ["CVE-2020-0001"],
          installed_paths: ["/usr/lib/package"],
          bundle_identifier: "",
          signature_information: undefined,
        },
      ],
    });

    render(
      <InventoryVersionsModal hostSoftware={mockSoftware} onExit={jest.fn()} />
    );

    expect(screen.queryByText("Last opened")).not.toBeInTheDocument();
    expect(screen.queryByText("Never")).not.toBeInTheDocument();
  });
});
