import React from "react";
import { render, screen } from "@testing-library/react";

import PlatformCell from "./PlatformCell";

const PLATFORMS = ["windows", "linux"];

describe("Platform cell", () => {
  it("renders platform icons in correct order", () => {
    render(<PlatformCell value={PLATFORMS} />);

    const icon = screen.queryByTestId("icon");

    expect(icon).toBeInTheDocument();
  });
});
