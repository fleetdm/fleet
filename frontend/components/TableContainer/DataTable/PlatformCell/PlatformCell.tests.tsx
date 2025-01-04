import React from "react";
import { render, screen } from "@testing-library/react";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import { ScheduledQueryablePlatform } from "interfaces/platform";
import PlatformCell from "./PlatformCell";

const SCHEDULED_QUERYABLE_PLATFORMS: ScheduledQueryablePlatform[] = [
  "windows",
  "darwin",
  "linux",
];

describe("Platform cell", () => {
  it("renders platform icons in correct order", () => {
    render(<PlatformCell platforms={SCHEDULED_QUERYABLE_PLATFORMS} />);

    const icons = screen.queryByTestId("icons");
    const appleIcon = screen.queryByTestId("darwin-icon");
    const linuxIcon = screen.queryByTestId("linux-icon");
    const windowsIcon = screen.queryByTestId("windows-icon");

    expect(icons?.firstChild).toBe(appleIcon);
    expect(icons?.firstChild?.nextSibling).toBe(windowsIcon);
    expect(icons?.firstChild?.nextSibling?.nextSibling).toBe(linuxIcon);
  });
  it("renders all platforms targeted when no platforms passed in and scheduled", () => {
    render(<PlatformCell platforms={[]} />);

    const icons = screen.queryByTestId("icons");
    const appleIcon = screen.queryByTestId("darwin-icon");
    const linuxIcon = screen.queryByTestId("linux-icon");
    const windowsIcon = screen.queryByTestId("windows-icon");

    expect(icons?.firstChild).toBe(appleIcon);
    expect(icons?.firstChild?.nextSibling).toBe(windowsIcon);
    expect(icons?.firstChild?.nextSibling?.nextSibling).toBe(linuxIcon);
  });
});
