import React from "react";
import { render, screen } from "@testing-library/react";

import InstallerActionCell from "./InstallerActionCell";

describe("Issue cell", () => {
  it("renders add button if installer is available", async () => {
    render(<InstallerActionCell value={{ id: 1, platform: "darwin" }} />);

    expect(screen.getByText(/add/i)).toBeInTheDocument();
  });
  it("renders --- if installer is unavailable", async () => {
    render(<InstallerActionCell />);

    expect(screen.getByText(/---/i)).toBeInTheDocument();
  });
  it("renders checkmark if installer is already added", async () => {
    render(
      <InstallerActionCell
        value={{ id: 1, platform: "darwin", software_title_id: 1 }}
      />
    );

    const icon = screen.getByTestId("success-icon");
    expect(icon).toBeInTheDocument();
  });
});
