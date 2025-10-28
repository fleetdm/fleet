import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import {
  createMockSoftwarePackage,
  createMockSoftwareTitle,
} from "__mocks__/softwareMock";
import EditIconModal from "./EditIconModal";

const software = createMockSoftwareTitle();
const softwarePackage = createMockSoftwarePackage();
const MOCK_PROPS = {
  softwareId: 123,
  teamIdForApi: 456,
  software: softwarePackage,
  onExit: jest.fn(),
  refetchSoftwareTitle: jest.fn(),
  iconUploadedAt: "2025-09-03T12:00:00Z",
  setIconUploadedAt: jest.fn(),
  installerType: "package" as "package" | "vpp",
  previewInfo: {
    type: "apps",
    versions: software.versions?.length,
    source: software.source,
    currentIconUrl: null,
    name: software.name,
    titleName: software.name,
    countsUpdatedAt: "2025-09-03T12:00:00Z",
  },
};

describe("EditIconModal", () => {
  it("renders with the correct modal title for package, FileUploader, Preview tabs, save button", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<EditIconModal {...MOCK_PROPS} />);

    expect(screen.getByText(/edit package/i)).toBeInTheDocument();
    expect(screen.getByText("Choose file")).toBeInTheDocument();
    expect(screen.getByText("Preview")).toBeInTheDocument();
    expect(screen.getByText("Fleet")).toBeInTheDocument();
    expect(screen.getByText("Self-service")).toBeInTheDocument();
    const save = screen.getByRole("button", { name: "Save" });
    expect(save).toBeInTheDocument();
  });

  it("shows the correct software name and preview info in Fleet card", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<EditIconModal {...MOCK_PROPS} />);
    expect(screen.getAllByText(software.name).length).toBeGreaterThan(0);
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Vulnerabilities")).toBeInTheDocument();
    expect(screen.getByText("88.0.1")).toBeInTheDocument();
    expect(screen.getByText("20 vulnerabilities")).toBeInTheDocument();
  });

  it("calls onExit handler when modal close is triggered", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<EditIconModal {...MOCK_PROPS} />);

    await user.keyboard("{Escape}");

    expect(MOCK_PROPS.onExit).toHaveBeenCalled();
  });

  // Note: Rely on QA Wolf for E2e testing of file upload, preview, save, and remove icon
});
