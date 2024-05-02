import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import createMockPolicy from "__mocks__/policyMock";
import PoliciesTable from "./PoliciesTable";

describe("Policies table", () => {
  const testCriticalPolicy = createMockPolicy({ critical: true });

  it("Renders a tooltip including 'Premium feature' copy for a critical policy in Sandbox mode", () => {
    render(
      <PoliciesTable
        policiesList={[testCriticalPolicy]}
        isLoading={false}
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        onDeletePolicyClick={() => {}}
        currentTeam={{ id: -1, name: "All teams" }}
        isPremiumTier
        isSandboxMode
        searchQuery=""
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
      />
    );

    expect(
      screen.getByText("This policy has been marked as critical.", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(
      screen.getByText("This is a premium feature.", { exact: false })
    ).toBeInTheDocument();
  });

  it("Renders a tooltip excluding 'Premium feature' copy for a critical policy not in Sandbox mode", () => {
    render(
      <PoliciesTable
        policiesList={[testCriticalPolicy]}
        isLoading={false}
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        onDeletePolicyClick={() => {}}
        currentTeam={{ id: -1, name: "All teams" }}
        isPremiumTier
        isSandboxMode={false}
        searchQuery=""
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
      />
    );

    expect(
      screen.getByText("This policy has been marked as critical.", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(
      screen.queryByText("This is a premium feature.", { exact: false })
    ).toBeNull();
  });
});
