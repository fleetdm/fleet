import React from "react";

import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import IssuesIndicator from "./IssuesIndicator";

describe("Issues indicator", () => {
  it("renders total issues count, critical vulnerabilities count, and failing policies count", async () => {
    const { user } = renderWithSetup(
      <IssuesIndicator
        totalIssuesCount={5}
        criticalVulnerabilitiesCount={3}
        failingPoliciesCount={2}
      />
    );
    await user.hover(screen.getByText("5"));

    const vulnerabilitiesTooltip = screen.getByText(
      /Critical vulnerabilities/i
    );
    const policiesTooltip = screen.getByText(/Failing policies/i);

    expect(vulnerabilitiesTooltip).toBeInTheDocument();
    expect(policiesTooltip).toBeInTheDocument();
  });
});
