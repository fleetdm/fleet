import React from "react";
import { render, screen } from "@testing-library/react";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import { QueryablePlatform } from "interfaces/platform";
import PlatformCell from "./PlatformCell";

const PLATFORMS: QueryablePlatform[] = ["windows", "darwin", "linux", "chrome"];

describe("Platform cell", () => {
  it("renders platform icons in correct order", () => {
    render(<PlatformCell platforms={PLATFORMS} />);

    const icons = screen.queryByTestId("icons");
    const appleIcon = screen.queryByTestId("darwin-icon");
    const linuxIcon = screen.queryByTestId("linux-icon");
    const windowsIcon = screen.queryByTestId("windows-icon");
    const chromeIcon = screen.queryByTestId("chrome-icon");

    expect(icons?.firstChild).toBe(appleIcon);
    expect(icons?.firstChild?.nextSibling).toBe(windowsIcon);
    expect(icons?.firstChild?.nextSibling?.nextSibling).toBe(linuxIcon);
    expect(icons?.firstChild?.nextSibling?.nextSibling?.nextSibling).toBe(
      chromeIcon
    );
  });
  it("renders empty state", () => {
    render(<PlatformCell platforms={[]} />);

    const emptyText = screen.queryByText(DEFAULT_EMPTY_CELL_VALUE);

    expect(emptyText).toBeInTheDocument();
  });
});
