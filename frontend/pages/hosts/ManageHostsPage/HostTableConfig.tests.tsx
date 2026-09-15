import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { generateAvailableTableHeaders } from "./HostTableConfig";

describe("HostTableConfig - Serial number column", () => {
  const headers = generateAvailableTableHeaders({
    isFreeTier: false,
    isOnlyObserver: false,
  });

  const serialColumn = headers.find((h) => h.id === "hardware_serial") as any;

  if (!serialColumn || typeof serialColumn.Cell !== "function") {
    throw new Error("hardware_serial column or Cell not found");
  }

  const Cell = serialColumn.Cell as React.ElementType;

  const renderCell = (
    serial: string,
    platform: string,
    mdm?: { enrollment_status: string }
  ) =>
    render(
      <Cell
        cell={{ value: serial }}
        row={{
          original: {
            platform,
            hardware_serial: serial,
            mdm,
          },
        }}
      />
    );

  it("shows the serial number for a macOS host", () => {
    renderCell("ABC123", "darwin", { enrollment_status: "On (automatic)" });
    expect(screen.getByText("ABC123")).toBeInTheDocument();
    expect(screen.queryByText("Not supported")).not.toBeInTheDocument();
  });

  it("shows the serial number for a managed Android host", () => {
    renderCell("PIXEL10A", "android", { enrollment_status: "On (automatic)" });
    expect(screen.getByText("PIXEL10A")).toBeInTheDocument();
    expect(screen.queryByText("Not supported")).not.toBeInTheDocument();
  });

  it("shows the serial number for an Android host with no mdm data", () => {
    // Regression guard: the cell must not crash dereferencing a missing `mdm`.
    renderCell("PIXEL10A", "android", undefined);
    expect(screen.getByText("PIXEL10A")).toBeInTheDocument();
    expect(screen.queryByText("Not supported")).not.toBeInTheDocument();
  });

  it("shows the serial number for a managed (ADE) iPadOS host", () => {
    renderCell("IPAD123", "ipados", { enrollment_status: "On (automatic)" });
    expect(screen.getByText("IPAD123")).toBeInTheDocument();
    expect(screen.queryByText("Not supported")).not.toBeInTheDocument();
  });

  it("shows 'Not supported' for a personal (BYOD) Android host", () => {
    renderCell("", "android", { enrollment_status: "On (manual - personal)" });
    expect(screen.getByText("Not supported")).toBeInTheDocument();
  });

  it("shows 'Not supported' for a personal (BYOD) iOS host", () => {
    renderCell("", "ios", { enrollment_status: "On (manual - personal)" });
    expect(screen.getByText("Not supported")).toBeInTheDocument();
  });
});

describe("HostTableConfig - Status column", () => {
  it("renders '---' with the Windows Autopilot tooltip for a pending Windows host", async () => {
    const statusColumn = generateAvailableTableHeaders({
      isFreeTier: false,
      isOnlyObserver: false,
    }).find((h) => h.id === "status") as any;
    const Cell = statusColumn.Cell as React.ElementType;
    const user = userEvent.setup();

    render(
      <Cell
        cell={{ value: "offline" }}
        row={{
          original: {
            platform: "windows",
            mdm: { enrollment_status: "Pending" },
          },
        }}
      />
    );

    await user.hover(screen.getByText("---"));

    expect(
      await screen.findByText(
        "Device is pending enrollment in Windows Autopilot and status is not yet available."
      )
    ).toBeInTheDocument();
  });
});
