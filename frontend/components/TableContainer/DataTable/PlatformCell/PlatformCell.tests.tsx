import React from "react";
import { render, screen } from "@testing-library/react";

import { QueryablePlatform } from "interfaces/platform";
import PlatformCell from "./PlatformCell";

const QUERYABLE_PLATFORMS: QueryablePlatform[] = [
  "windows",
  "darwin",
  "linux",
  "chrome",
];

describe("Platform cell", () => {
  it("renders platform icons in correct order", () => {
    render(<PlatformCell platforms={QUERYABLE_PLATFORMS} />);

    const icons = screen.queryByTestId("icons");
    const appleIcon = screen.queryByTestId("darwin-icon");
    const windowsIcon = screen.queryByTestId("windows-icon");
    const linuxIcon = screen.queryByTestId("linux-icon");
    const chromeIcon = screen.queryByTestId("chrome-icon");

    expect(icons?.firstChild).toBe(appleIcon);
    expect(icons?.firstChild?.nextSibling).toBe(windowsIcon);
    expect(icons?.firstChild?.nextSibling?.nextSibling).toBe(linuxIcon);
    expect(icons?.firstChild?.nextSibling?.nextSibling?.nextSibling).toBe(
      chromeIcon
    );
  });
  it("renders --- when no platforms passed in", () => {
    render(<PlatformCell platforms={[]} />);

    expect(screen.getByText(/---/i)).toBeInTheDocument();
  });
});
