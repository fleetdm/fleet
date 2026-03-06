import React from "react";
import { screen, render } from "@testing-library/react";
import { ISoftwareInstallerPolicyIncludeType } from "interfaces/software";
import InstallerPoliciesTable from "./InstallerPoliciesTable";

describe("InstallerPoliciesTable", () => {
  it("renders policy names as links and footer info", () => {
    const policies: ISoftwareInstallerPolicyIncludeType[] = [
      { id: 1, name: "No Gatekeeper", type: new Set(["dynamic"]) },
      { id: 2, name: "Outdated Gatekeeper", type: new Set(["patch"]) },
    ];

    render(<InstallerPoliciesTable teamId={42} policies={policies} />);

    // There should be two cells, each with a link
    const cells = screen.getAllByRole("cell");
    expect(cells).toHaveLength(2);

    // Each cell should contain a link with the policy name
    expect(cells[0].querySelector("a.link-cell")).toHaveTextContent(
      /No Gatekeeper/i
    );
    expect(cells[0].querySelector("a.link-cell")).toHaveTextContent(
      /Outdated Gatekeeper/i
    );
    expect(screen.getByText(/2 policies/i)).toBeInTheDocument();

    // TODO: Update to have hover on "refresh-icon" for the auto install one and hover on word "patch" for patch text
  });
});
