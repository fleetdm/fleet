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

  const Cell = lastOpenedColumn.Cell as React.ElementType;

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
