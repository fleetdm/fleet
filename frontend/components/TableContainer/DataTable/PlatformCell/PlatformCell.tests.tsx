import React from "react";
import { render, screen } from "@testing-library/react";

import PlatformCell from "./PlatformCell";

const PLATFORMS = ["windows", "darwin", "linux"];

describe("Platform cell", () => {
  it("renders platform icons in correct order", () => {
    render(<PlatformCell value={PLATFORMS} />);

    const icons = screen.queryAllByTestId("icon");
    const appleIcon = screen.queryByTestId("apple-icon");
    const linuxIcon = screen.queryByTestId("linux-icon");
    const windowsIcon = screen.queryByTestId("windows-icon");

    expect(icons).toHaveLength(3);
    expect(icons[0].firstChild).toBe(appleIcon);
    expect(icons[1].firstChild).toBe(linuxIcon);
    expect(icons[2].firstChild).toBe(windowsIcon);
  });
  it("renders empty state", () => {
    render(<PlatformCell value={[]} />);

    const icons = screen.queryAllByTestId("icon");
    const emptyText = screen.queryByText("---");

    expect(icons).toHaveLength(0);
    expect(emptyText).toBeInTheDocument();
  });
});
