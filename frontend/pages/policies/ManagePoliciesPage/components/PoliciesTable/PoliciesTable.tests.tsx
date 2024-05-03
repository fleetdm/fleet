import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import createMockPolicy from "__mocks__/policyMock";
import PoliciesTable from "./PoliciesTable";

describe("Policies table", () => {
  it("Renders the page-wide empty state when no policies are present", async () => {
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

    expect(screen.getByText("You don't have any policies")).toBeInTheDocument();
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

    const { user } = render(
      <PoliciesTable
        policiesList={[]}
        isLoading={false}
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        onDeletePolicyClick={() => {}}
        currentTeam={{ id: -1, name: "All teams" }}
        isPremiumTier
        isSandboxMode={false}
        searchQuery="shouldn't match anything"
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
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
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        onDeletePolicyClick={() => {}}
        currentTeam={{ id: 2, name: "Team 2" }}
        isPremiumTier
        isSandboxMode={false}
        searchQuery=""
        page={0}
        onQueryChange={noop}
        renderPoliciesCount={() => null}
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

    expect(screen.queryByText("Inherited")).not.toBeInTheDocument();
  });
});
