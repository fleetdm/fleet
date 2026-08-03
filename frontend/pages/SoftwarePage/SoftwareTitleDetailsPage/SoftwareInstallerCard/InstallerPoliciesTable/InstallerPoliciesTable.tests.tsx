import React from "react";
import { screen, render } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import { ISoftwareInstallPolicyUI } from "interfaces/software";
import InstallerPoliciesTable from "./InstallerPoliciesTable";

describe("InstallerPoliciesTable", () => {
  const policies: ISoftwareInstallPolicyUI[] = [
    { id: 1, name: "No Gatekeeper", type: new Set(["dynamic"]) },
    { id: 2, name: "Outdated Gatekeeper", type: new Set(["patch"]) },
  ];
  it("renders policy names as links and footer info", () => {
    render(<InstallerPoliciesTable teamId={42} policies={policies} />);

    // There should be two cells, each with a link
    const cells = screen.getAllByRole("cell");
    expect(cells).toHaveLength(2);

    // Each cell should contain a link with the policy name
    expect(cells[0].querySelector("a.link-cell")).toHaveTextContent(
      /No Gatekeeper/i
    );
    expect(cells[1].querySelector("a.link-cell")).toHaveTextContent(
      /Outdated Gatekeeper/i
    );
    expect(screen.getByText(/2 policies/i)).toBeInTheDocument();
  });
  it("renders the badges for patch and dynamic policies", async () => {
    const { user } = renderWithSetup(
      <InstallerPoliciesTable teamId={42} policies={policies} />
    );

    expect(screen.getByTestId("refresh-icon")).toBeInTheDocument();

    await user.hover(screen.getByTestId("refresh-icon"));
    expect(
      await screen.findByText(
        "Software will be automatically installed when hosts fail this policy."
      )
    ).toBeInTheDocument();

    await user.hover(screen.getByText(/patch/i));
    expect(
      await screen.findByText(
        "Hosts will fail this policy if they're running an older version."
      )
    ).toBeInTheDocument();
  });
});
