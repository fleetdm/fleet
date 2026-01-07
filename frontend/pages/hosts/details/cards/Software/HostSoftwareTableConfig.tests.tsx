import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import { createMockRouter } from "test/test-utils";

import { generateSoftwareTableHeaders } from "./HostSoftwareTableConfig";

const mockRouter = createMockRouter();

describe("HostSoftwareTableConfig - Last opened column", () => {
  const headers = generateSoftwareTableHeaders({
    router: mockRouter,
    teamId: 1,
    onShowInventoryVersions: noop,
    platform: "windows",
  });

  const lastOpenedColumn = headers.find((h) => h.id === "Last opened") as any;

  if (!lastOpenedColumn || typeof lastOpenedColumn.accessor !== "function") {
    throw new Error("Last opened column or accessor not found");
  }

  const accessor = lastOpenedColumn.accessor;
  const Cell = lastOpenedColumn.Cell as React.FC<any>;

  describe("accessor", () => {
    it("returns a date string if at least one version has a valid date", () => {
      const mockSoftware: any = {
        installed_versions: [
          { version: "1.0", last_opened_at: "2023-01-01T00:00:00Z" },
          { version: "1.1", last_opened_at: "" },
        ],
      };
      expect(accessor(mockSoftware, 0, [])).toBe("2023-01-01T00:00:00Z");
    });

    it("returns the most recent date string if multiple versions have dates", () => {
      const mockSoftware: any = {
        installed_versions: [
          { version: "1.0", last_opened_at: "2023-01-01T00:00:00Z" },
          { version: "1.1", last_opened_at: "2023-05-01T00:00:00Z" },
        ],
      };
      expect(accessor(mockSoftware, 0, [])).toBe("2023-05-01T00:00:00Z");
    });

    it("returns an empty string if supported but no versions have been opened", () => {
      const mockSoftware: any = {
        installed_versions: [
          { version: "1.0", last_opened_at: "" },
          { version: "1.1", last_opened_at: "" },
        ],
      };
      expect(accessor(mockSoftware, 0, [])).toBe("");
    });

    it("returns undefined if not supported (last_opened_at is missing from all versions)", () => {
      const mockSoftware: any = {
        installed_versions: [{ version: "1.0" }, { version: "1.1" }],
      };
      expect(accessor(mockSoftware, 0, [])).toBeUndefined();
    });
  });

  describe("Cell", () => {
    it("renders the date when a valid date string is provided", () => {
      render(
        <Cell cell={{ value: "2023-01-01T00:00:00Z" }} row={{ original: {} }} />
      );
      // HumanTimeDiffWithDateTip will render something like "2 years ago" or similar depending on current date
      // but it definitely won't be "Never" or "Not supported"
      expect(screen.queryByText("Never")).not.toBeInTheDocument();
      expect(screen.queryByText("Not supported")).not.toBeInTheDocument();
    });

    it("renders 'Never' when the value is an empty string", () => {
      render(<Cell cell={{ value: "" }} row={{ original: {} }} />);
      expect(screen.getByText("Never")).toBeInTheDocument();
    });

    it("renders 'Not supported' when the value is undefined", () => {
      render(<Cell cell={{ value: undefined }} row={{ original: {} }} />);
      expect(screen.getByText("Not supported")).toBeInTheDocument();
    });
  });
});
