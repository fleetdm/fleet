import React from "react";
import { render, screen } from "@testing-library/react";

import createMockHost from "__mocks__/hostMock";
import { MdmEnrollmentStatus } from "interfaces/mdm";
import HostMdmStatusCell from "./HostMdmStatusCell";

const renderCell = (platform: string, value?: MdmEnrollmentStatus) => {
  const host = createMockHost({ platform } as any);
  return render(
    <HostMdmStatusCell row={{ original: host }} cell={{ value } as any} />
  );
};

describe("HostMdmStatusCell", () => {
  it("renders 'Not supported' for Chrome hosts", () => {
    renderCell("chrome");
    expect(screen.getByText("Not supported")).toBeInTheDocument();
  });

  it("renders 'Not supported' for Linux hosts", () => {
    renderCell("ubuntu");
    expect(screen.getByText("Not supported")).toBeInTheDocument();
  });

  it("renders 'On (manual)' for Apple hosts with manual enrollment", () => {
    renderCell("darwin", "On (manual)");
    expect(screen.getByText("On (manual)")).toBeInTheDocument();
  });

  it("renders 'On (company-owned)' for Apple hosts with automatic enrollment", () => {
    renderCell("darwin", "On (automatic)");
    expect(screen.getByText("On (company-owned)")).toBeInTheDocument();
  });

  it("renders 'On (manual - personal)' for iOS hosts with personal enrollment", () => {
    renderCell("ios", "On (manual - personal)");
    expect(screen.getByText("On (manual - personal)")).toBeInTheDocument();
  });

  it("renders 'Pending' for macOS hosts with pending enrollment", () => {
    renderCell("darwin", "Pending");
    expect(screen.getByText("Pending")).toBeInTheDocument();
  });

  it("renders the MDM status for Android hosts", () => {
    renderCell("android", "On (manual - personal)");
    expect(screen.getByText("On (manual - personal)")).toBeInTheDocument();
  });

  it("renders the MDM status for Windows hosts", () => {
    renderCell("windows", "On (manual)");
    expect(screen.getByText("On (manual)")).toBeInTheDocument();
  });

  it("renders 'Off' for Windows hosts with no MDM enrollment", () => {
    renderCell("windows", "Off");
    expect(screen.getByText("Off")).toBeInTheDocument();
  });
});
