import { render, screen } from "@testing-library/react";
import React from "react";

import OrgNameCell from "./OrgNameCell";

describe("OrgNameCell", () => {
  it("renders the org name without a default tag for a non-default token", () => {
    render(
      <OrgNameCell orgName="Acme Inc." termsExpired={false} isDefault={false} />
    );

    expect(screen.getByText("Acme Inc.")).toBeInTheDocument();
    expect(screen.queryByText("Default token")).not.toBeInTheDocument();
  });

  it("renders the default tag for the default token", () => {
    render(<OrgNameCell orgName="Acme Inc." termsExpired={false} isDefault />);

    expect(screen.getByText("Acme Inc.")).toBeInTheDocument();
    expect(screen.getByText("Default token")).toBeInTheDocument();
  });

  it("renders the default tag alongside the expired terms warning", () => {
    render(<OrgNameCell orgName="Acme Inc." termsExpired isDefault />);

    expect(screen.getByText("Acme Inc.")).toBeInTheDocument();
    expect(screen.getByText("Default token")).toBeInTheDocument();
  });
});
