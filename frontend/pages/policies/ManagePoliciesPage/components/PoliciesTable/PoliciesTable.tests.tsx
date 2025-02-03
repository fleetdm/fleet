import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import createMockPolicy from "__mocks__/policyMock";
import PoliciesTable from "./PoliciesTable";

describe("Policies table", () => {
  it("Renders the page-wide empty state when no policies are present (all teams)", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <PoliciesTable
        policiesList={[]}
        isLoading={false}
        onDeletePolicyClick={noop}
        currentTeam={{ id: -1, name: "All teams" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
        resetPageIndex={false}
      />
    );

    expect(
      screen.getByText("You don't have any policies that apply to all teams")
    ).toBeInTheDocument();
    expect(screen.queryByText("Name")).toBeNull();
  });

  it("Renders the page-wide empty state when no policies are present (specific team)", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <PoliciesTable
        policiesList={[]}
        isLoading={false}
        onDeletePolicyClick={noop}
        currentTeam={{ id: 1, name: "Some team" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
        resetPageIndex={false}
      />
    );

    expect(
      screen.getByText("You don't have any policies that apply to this team")
    ).toBeInTheDocument();
    expect(screen.queryByText("Name")).toBeNull();
  });

  it("Renders the empty search state when search query exists for server side search with no results", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <PoliciesTable
        policiesList={[]}
        isLoading={false}
        onDeletePolicyClick={noop}
        currentTeam={{ id: -1, name: "All teams" }}
        isPremiumTier
        searchQuery="shouldn't match anything"
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
        resetPageIndex={false}
      />
    );

    expect(screen.getByText("No matching policies")).toBeInTheDocument();
    expect(screen.queryByText("Name")).toBeNull();
  });

  it("Renders a critical badge and tooltip for a critical policy", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const testCriticalPolicy = createMockPolicy({ critical: true });

    const { user } = render(
      <PoliciesTable
        policiesList={[testCriticalPolicy]}
        isLoading={false}
        onDeletePolicyClick={noop}
        currentTeam={{ id: -1, name: "All teams" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
        resetPageIndex={false}
      />
    );

    await waitFor(() => {
      waitFor(() => {
        user.hover(screen.getByTestId("policy-icon"));
      });

      expect(
        screen.getByText("This policy has been marked as critical.")
      ).toBeInTheDocument();
    });
  });

  it("Renders an inherited badge and tooltip for inherited policy on a team's policies page", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const testInheritedPolicy = createMockPolicy({ team_id: null });

    const { user } = render(
      <PoliciesTable
        policiesList={[testInheritedPolicy]}
        isLoading={false}
        onDeletePolicyClick={noop}
        currentTeam={{ id: 2, name: "Team 2" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
        resetPageIndex={false}
      />
    );

    await waitFor(() => {
      waitFor(() => {
        user.hover(screen.getByText("Inherited"));
      });

      expect(
        screen.getByText("This policy runs on all hosts.")
      ).toBeInTheDocument();
    });
  });

  it("Does not render an inherited badge and tooltip for global policy on the All teams's policies page", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const testGlobalPolicy = createMockPolicy({ team_id: null });

    render(
      <PoliciesTable
        policiesList={[testGlobalPolicy]}
        isLoading={false}
        onDeletePolicyClick={noop}
        currentTeam={{ id: -1, name: "All teams" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
        resetPageIndex={false}
      />
    );

    expect(screen.queryByText("Inherited")).not.toBeInTheDocument();
  });

  it("Renders the correct number of checkboxes for team policies and not inherited policies on a team's policies page and can check select all box", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const testInheritedPolicies = [
      createMockPolicy({ team_id: null, name: "Inherited policy 1" }),
      createMockPolicy({ id: 2, team_id: null, name: "Inherited policy 2" }),
      createMockPolicy({ id: 3, team_id: null, name: "Inherited policy 3" }),
    ];

    const testTeamPolicies = [
      createMockPolicy({ id: 4, team_id: 2, name: "Team policy 1" }),
      createMockPolicy({ id: 5, team_id: 2, name: "Team policy 2" }),
    ];

    const { container, user } = render(
      <PoliciesTable
        policiesList={[...testInheritedPolicies, ...testTeamPolicies]}
        isLoading={false}
        onDeletePolicyClick={noop}
        currentTeam={{ id: 2, name: "Team 2" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
        canAddOrDeletePolicy
        hasPoliciesToDelete
        resetPageIndex={false}
      />
    );

    const numberOfCheckboxes = container.querySelectorAll(
      "input[type='checkbox']"
    ).length;

    expect(numberOfCheckboxes).toBe(
      testTeamPolicies.length + 1 // +1 for Select all checkbox
    );

    const checkbox = container.querySelectorAll(
      "input[type='checkbox']"
    )[0] as HTMLInputElement;

    await waitFor(() => {
      waitFor(() => {
        user.click(checkbox);
      });

      expect(checkbox.checked).toBe(true);
    });
  });
});
