import { render, screen } from "@testing-library/react";
import React from "react";

import ApiEndpointCountTag from "./ApiEndpointCountTag";

describe("ApiEndpointCountTag", () => {
  it("pluralizes the endpoint count", () => {
    render(<ApiEndpointCountTag count={16} />);

    expect(screen.getByText("16 API endpoints")).toBeInTheDocument();
  });

  it("singularizes a count of one", () => {
    render(<ApiEndpointCountTag count={1} />);

    expect(screen.getByText("1 API endpoint")).toBeInTheDocument();
  });
});
