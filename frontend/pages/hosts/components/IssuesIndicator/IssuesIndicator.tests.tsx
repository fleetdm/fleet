import React from "react";

import { screen, render, fireEvent } from "@testing-library/react";

import IssuesIndicator from "./IssuesIndicator";

describe("Issues indicator", () => {
  it("renders total issues count, critical vulnerabilities count, and failing policies count", async () => {
    render(
      <IssuesIndicator
        totalIssuesCount={5}
        criticalVulnerabilitiesCount={3}
        failingPoliciesCount={2}
      />
    );
    await fireEvent.mouseOver(screen.getByText("5"));

    const vulnerabilitiesTooltip = screen.getByText(
      /Critical vulnerabilities/i
    );
    const policiesTooltip = screen.getByText(/Failing policies/i);

    expect(vulnerabilitiesTooltip).toBeInTheDocument();
    expect(policiesTooltip).toBeInTheDocument();
  });
});
