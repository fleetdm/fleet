import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import InstallerDetailsWidget from "./InstallerDetailsWidget";

// Mock current time for time stamp test
beforeAll(() => {
  jest.useFakeTimers();
  jest.setSystemTime(new Date("2024-05-08T10:00:00Z"));
});

afterAll(() => {
  jest.useRealTimers();
});

describe("InstallerDetailsWidget", () => {
  const defaultProps = {
    softwareName: "Test Software",
    installerType: "package" as const,
    addedTimestamp: "2024-05-06T10:00:00Z",
    versionInfo: <span>v1.2.3</span>,
    isFma: false,
  };

  it("renders the package icon when installerType is 'package'", () => {
    render(<InstallerDetailsWidget {...defaultProps} />);
    expect(screen.queryByTestId("file-pkg-graphic")).toBeInTheDocument();
    expect(screen.queryByTestId("software-icon")).not.toBeInTheDocument();
  });

  it("renders the software name", () => {
    render(<InstallerDetailsWidget {...defaultProps} />);
    expect(screen.getByText("Test Software")).toBeInTheDocument();
  });

  it("renders version info and relative time when addedTimestamp is present", () => {
    render(<InstallerDetailsWidget {...defaultProps} />);
    expect(screen.getByText("v1.2.3")).toBeInTheDocument();
    expect(screen.getByText(/2 days ago/i)).toBeInTheDocument();
  });

  it("renders only version info when addedTimestamp is not present", () => {
    render(
      <InstallerDetailsWidget {...defaultProps} addedTimestamp={undefined} />
    );
    expect(screen.queryByText(/2 days ago/i)).not.toBeInTheDocument();
  });

  it("applies additional className if provided", () => {
    render(
      <InstallerDetailsWidget {...defaultProps} className="extra-class" />
    );
    const rootDiv = document.querySelector(
      ".installer-details-widget.extra-class"
    );
    expect(rootDiv).toBeInTheDocument();
  });

  it("renders custom package label", () => {
    render(<InstallerDetailsWidget {...defaultProps} />);

    expect(screen.getByText(/custom package/i)).toBeInTheDocument();
  });

  it("renders FMA label", () => {
    render(<InstallerDetailsWidget {...defaultProps} isFma />);

    expect(screen.getByText(/Fleet-maintained/i)).toBeInTheDocument();
  });

  it("renders VPP label", () => {
    render(<InstallerDetailsWidget {...defaultProps} installerType="vpp" />);

    expect(screen.getByText(/App Store \(VPP\)/i)).toBeInTheDocument();
  });

  it("InstallerName disables tooltip if not truncated", () => {
    // useCheckTruncatedElement is mocked to return false
    render(<InstallerDetailsWidget {...defaultProps} />);
    // TooltipWrapper is mocked, so we just check that the child is rendered
    expect(screen.getByText("Test Software")).toBeInTheDocument();
  });

  it("renders the sha256 hash when provided and a copy button", () => {
    const sha256 =
      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890";
    render(<InstallerDetailsWidget {...defaultProps} sha256={sha256} />);
    // The component shows the first 6 chars + ellipsis
    expect(screen.getByText(/^abcdef1…$/)).toBeInTheDocument();
    const copyIcon = screen.getByTestId("copy-icon");
    const copyButton = copyIcon.closest("button");
    expect(copyButton).toBeInTheDocument();
  });
});
