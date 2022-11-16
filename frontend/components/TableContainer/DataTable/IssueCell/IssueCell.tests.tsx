import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import IssueCell from "./IssueCell";

describe("Issue cell", () => {
  it("renders icon, total issues, and failing policies tooltip", async () => {
    const { user } = renderWithSetup(
      <IssueCell
        issues={{
          total_issues_count: 4,
          failing_policies_count: 2,
        }}
        rowId={1}
      />
    );

    const icon = screen.queryByTestId("icon");

    await user.hover(screen.getByText("4"));

    expect(screen.getByText(/failing policies/i)).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });
});
