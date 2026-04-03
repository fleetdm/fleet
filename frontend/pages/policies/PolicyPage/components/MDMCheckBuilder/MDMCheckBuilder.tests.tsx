import React from "react";

import { render, screen } from "@testing-library/react";

import MDMCheckBuilder from "./MDMCheckBuilder";

describe("MDMCheckBuilder component", () => {
  it("renders the add check button", () => {
    render(
      <MDMCheckBuilder checks={[]} onChange={() => undefined} />
    );
    expect(screen.getByText(/add check/i)).toBeInTheDocument();
  });
});
