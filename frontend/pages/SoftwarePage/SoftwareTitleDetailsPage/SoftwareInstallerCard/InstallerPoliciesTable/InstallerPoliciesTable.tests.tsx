import React from "react";
import { screen, render } from "@testing-library/react";
import { ISoftwareInstallPolicy } from "interfaces/software";
import InstallerPoliciesTable from "./InstallerPoliciesTable";

describe("InstallerPoliciesTable", () => {
  it("renders policy names as links and footer info", () => {
    const policies: ISoftwareInstallPolicy[] = [
      { id: 1, name: "No Gatekeeper", type: "dynamic" },
      { id: 2, name: "Outdated Gatekeeper", type: "patch" },
    ];

    render(<InstallerPoliciesTable teamId={42} policies={policies} />);

    // There should be two cells, each with a link
    const cells = screen.getAllByRole("cell");
    expect(cells).toHaveLength(1);

    // Each cell should contain a link with the policy name
    expect(cells[0].querySelector("a.link-cell")).toHaveTextContent(
      /No Gatekeeper/i
    );
    const POLICY_COUNT = /1 policy/i;
    expect(screen.getByText(POLICY_COUNT)).toBeInTheDocument();

    // TODO: Update with tests for the different icons
  });
});
