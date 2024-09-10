import React from "react";
import { render } from "@testing-library/react";

import PlatformIcon from "./PlatformIcon";

describe("PlatformIcon - component", () => {
  it("renders", () => {
    const { container } = render(<PlatformIcon name="linux" />);
    expect(container.querySelector(".platform-icon")).toBeInTheDocument();
  });

  it("renders text if no icon", () => {
    const { container } = render(<PlatformIcon name="All" />);
    expect(
      container.querySelector(".fleeticon-single-host")
    ).toBeInTheDocument();
  });
});
