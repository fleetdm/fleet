import React from "react";
import { render, screen } from "@testing-library/react";
import InventoryVersions from "./InventoryVersions";

describe("InventoryVersions component", () => {
  it("renders 'Never' when last_opened_at is an empty string", () => {
    const mockSoftware: any = {
      source: "apps",
      installed_versions: [
        {
          version: "1.0",
          last_opened_at: "",
        },
      ],
    };
    render(<InventoryVersions hostSoftware={mockSoftware} />);
    expect(screen.getByText("Never")).toBeInTheDocument();
  });

  it("renders 'Not supported' when last_opened_at is undefined", () => {
    const mockSoftware: any = {
      source: "chrome_extensions",
      installed_versions: [
        {
          version: "1.0",
          // last_opened_at is missing
        },
      ],
    };
    render(<InventoryVersions hostSoftware={mockSoftware} />);
    expect(screen.getByText("Not supported")).toBeInTheDocument();
  });

  it("renders the date ago when last_opened_at is a valid date", () => {
    const mockSoftware: any = {
      source: "apps",
      installed_versions: [
        {
          version: "1.0",
          last_opened_at: new Date().toISOString(),
        },
      ],
    };
    render(<InventoryVersions hostSoftware={mockSoftware} />);
    // dateAgo(now) should return something like "just now" or "1 minute ago"
    // but definitely not "Never" or "Not supported"
    expect(screen.queryByText("Never")).not.toBeInTheDocument();
    expect(screen.queryByText("Not supported")).not.toBeInTheDocument();
  });
});
