import React from "react";

import { render, screen } from "@testing-library/react";

import InfoBanner from "./InfoBanner";

describe("InfoBanner - component", () => {
  it("info banner renders text", () => {
    render(<InfoBanner>Info banner testing text</InfoBanner>);

    const title = screen.getByText("Info banner testing text");

    expect(title).toBeInTheDocument();
  });
});
