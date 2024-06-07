import React from "react";
import { render, screen } from "@testing-library/react";

import IssueCell from "./IssueCell";

describe("Issue cell", () => {
  it("renders --- if total issues count is 0", async () => {
    render(
      <IssueCell
        issues={{
          total_issues_count: 0,
          failing_policies_count: 0,
        }}
        rowId={1}
      />
    );

    expect(screen.getByText(/---/i)).toBeInTheDocument();
  });
});
