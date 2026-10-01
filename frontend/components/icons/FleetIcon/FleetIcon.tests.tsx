import { render } from "@testing-library/react";
import React from "react";

import FleetIcon from "./FleetIcon";

describe("FleetIcon - component", () => {
  it("renders", () => {
    const { container } = render(<FleetIcon name="success-check" />);
    expect(
      container.querySelector(".fleeticon-success-check")
    ).toBeInTheDocument();
  });
});
