import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostSoftware } from "__mocks__/hostMock";
import SoftwareDetailsModal from "./SoftwareDetailsModal";

// Mock current time for time stamp test
beforeAll(() => {
  jest.useFakeTimers();
  jest.setSystemTime(new Date("2022-05-08T10:00:00Z"));
});

describe("SoftwareDetailsModal", () => {
  it("renders details including hash, vulnerabilities, and paths", () => {
    const mockSoftware = createMockHostSoftware();
    render(
      <SoftwareDetailsModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        onExit={jest.fn()}
      />
    );

    // Modal title
    expect(screen.getByText(mockSoftware.name)).toBeVisible();

    // Version, Type, Bundle identifier, Last used
    expect(screen.getByText("Version")).toBeVisible();
    expect(screen.getByText("1.0.0")).toBeVisible();
    expect(screen.getByText("Type")).toBeVisible();
    expect(screen.getByText("Application (macOS)")).toBeVisible();
    expect(screen.getByText("Bundle identifier")).toBeVisible();
    expect(screen.getByText("com.test.mock")).toBeVisible();
    expect(screen.getByText("Last used")).toBeVisible();
    expect(screen.getByText("4 months ago")).toBeVisible();

    // File path
    expect(screen.getByText("File path")).toBeVisible();
    expect(screen.getByText("/Applications/mock.app")).toBeVisible();

    // Hash
    expect(screen.getByText("Hash")).toBeVisible();
    expect(screen.getByText("mockhashhere")).toBeVisible();

    // Vulnerabilities
    expect(screen.getByText(/CVE-2020-0001/)).toBeVisible();

    // Tabs: Software details and Install details
    expect(screen.getByText("Software details")).toBeVisible();
    expect(screen.getByText("Install details")).toBeVisible();

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
      <SoftwareDetailsModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        onExit={jest.fn()}
      />
    );

    expect(screen.queryByText("Hash")).not.toBeInTheDocument();
    expect(screen.queryByText("mockhashhere")).not.toBeInTheDocument();
  });

  it("renders only software type if there are no installed versions", () => {
    const mockSoftware = createMockHostSoftware({ installed_versions: [] });
    render(
      <SoftwareDetailsModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        onExit={jest.fn()}
      />
    );
    expect(screen.getByText("Type")).toBeVisible();
    expect(screen.getByText("Application (macOS)")).toBeVisible();
    expect(screen.queryByText("Version")).not.toBeInTheDocument();
    expect(screen.queryByText("File path")).not.toBeInTheDocument();
  });
});
