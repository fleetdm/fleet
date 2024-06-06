import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import IssueCell from "./IssueCell";

describe("Issue cell", () => {
  it("renders icon, total issues, and failing policies tooltip", async () => {
    const render = createCustomRenderer({});

    const { user } = render(
      <IssueCell
        issues={{
          total_issues_count: 4,
          critical_vulnerabilities_count: 2,
          failing_policies_count: 2,
        }}
        rowId={1}
      />
    );

    const icon = screen.queryByTestId("error-outline-icon");

    await user.hover(screen.getByText("4"));

    expect(screen.getByText(/failing policies/i)).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });
});
