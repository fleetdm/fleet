import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import createMockPolicy from "__mocks__/policyMock";
import PoliciesTable from "./PoliciesTable";

const mockRouter = createMockRouter();

describe("Policies table", () => {
  it("Renders the page-wide empty state when no policies are present (free tier)", async () => {
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
        onDeletePoliciesClick={noop}
        currentTeam={{ id: -1, name: "All fleets" }}
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={0}
      />
    );

    expect(screen.getByText("You don't have any policies")).toBeInTheDocument();
    expect(screen.queryByText("Name")).toBeNull();
    expect(screen.queryByPlaceholderText("Search by name")).toBeNull();
  });

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
        onDeletePoliciesClick={noop}
        currentTeam={{ id: -1, name: "All fleets" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={0}
      />
    );

    expect(
      screen.getByText("You don't have any policies that apply to all fleets")
    ).toBeInTheDocument();
    expect(screen.queryByText("Name")).toBeNull();
    expect(screen.queryByPlaceholderText("Search by name")).toBeNull();
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
        onDeletePoliciesClick={noop}
        currentTeam={{ id: 1, name: "Some team" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={0}
      />
    );

    expect(
      screen.getByText("You don't have any policies that apply to this fleet")
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
        onDeletePoliciesClick={noop}
        currentTeam={{ id: -1, name: "All fleets" }}
        isPremiumTier
        searchQuery="shouldn't match anything"
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={0}
      />
    );

    expect(screen.getByText("No matching policies")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Search by name")).toBeInTheDocument();
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
        onDeletePoliciesClick={noop}
        currentTeam={{ id: -1, name: "All fleets" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={[testCriticalPolicy].length}
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
        onDeletePoliciesClick={noop}
        currentTeam={{ id: 2, name: "Team 2" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={[testInheritedPolicy].length}
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

  it("Does not render an inherited badge and tooltip for global policy on the All fleets's policies page", () => {
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
        onDeletePoliciesClick={noop}
        currentTeam={{ id: -1, name: "All fleets" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={[testGlobalPolicy].length}
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

    const policiesList = [...testInheritedPolicies, ...testTeamPolicies];

    const { user } = render(
      <PoliciesTable
        policiesList={policiesList}
        isLoading={false}
        onDeletePoliciesClick={noop}
        currentTeam={{ id: 2, name: "Team 2" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        canAddOrDeletePolicies
        hasPoliciesToDelete
        count={policiesList.length}
      />
    );

    const numberOfCheckboxes = screen.queryAllByRole("checkbox").length;

    expect(numberOfCheckboxes).toBe(
      testTeamPolicies.length + 1 // +1 for Select all checkbox
    );

    const checkbox = screen.queryAllByRole("checkbox")[0];

    await waitFor(async () => {
      await user.click(checkbox);
    });

    expect(checkbox).toBeChecked();
  });

  it("Renders a Patch badge for a patch policy", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const testPatchPolicy = createMockPolicy({
      type: "patch",
      name: "macOS - Zoom up to date",
    });

    render(
      <PoliciesTable
        policiesList={[testPatchPolicy]}
        isLoading={false}
        onDeletePoliciesClick={noop}
        currentTeam={{ id: -1, name: "All fleets" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={1}
      />
    );

    expect(screen.getByText("Patch")).toBeInTheDocument();
  });

  it("Does not render a Patch badge for a dynamic policy", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const testDynamicPolicy = createMockPolicy({ type: "dynamic" });

    render(
      <PoliciesTable
        policiesList={[testDynamicPolicy]}
        isLoading={false}
        onDeletePoliciesClick={noop}
        currentTeam={{ id: -1, name: "All fleets" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={1}
      />
    );

    expect(screen.queryByText("Patch")).not.toBeInTheDocument();
  });

  it("Renders the Targeted platforms column using the policy's platform field", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const policyWithAllPlatforms = createMockPolicy({
      id: 100,
      name: "cross-platform policy",
      platform: "",
    });
    const policyWithDarwin = createMockPolicy({
      id: 101,
      name: "macOS policy",
      platform: "darwin",
    });

    render(
      <PoliciesTable
        policiesList={[policyWithAllPlatforms, policyWithDarwin]}
        isLoading={false}
        onDeletePoliciesClick={noop}
        currentTeam={{ id: -1, name: "All fleets" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={2}
      />
    );

    expect(screen.getByText("Targeted platforms")).toBeInTheDocument();
    expect(screen.getByTestId("darwin-icon")).toBeInTheDocument();
    expect(screen.queryByTestId("windows-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("linux-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("chrome-icon")).not.toBeInTheDocument();
  });

  it("Renders the platform filter dropdown when the table is searchable", () => {
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
        policiesList={[createMockPolicy({ platform: "darwin" })]}
        isLoading={false}
        onDeletePoliciesClick={noop}
        currentTeam={{ id: -1, name: "All fleets" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={1}
      />
    );

    expect(screen.getByText("All platforms")).toBeInTheDocument();
  });

  it("Renders the Automations column with correct values", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const policyWithAutomations = createMockPolicy({
      id: 10,
      name: "Policy with automations",
      install_software: { name: "Zoom", software_title_id: 1 },
      calendar_events_enabled: true,
    });

    const policyWithoutAutomations = createMockPolicy({
      id: 11,
      name: "Policy without automations",
      install_software: undefined,
      calendar_events_enabled: false,
      conditional_access_enabled: false,
    });

    render(
      <PoliciesTable
        policiesList={[policyWithAutomations, policyWithoutAutomations]}
        isLoading={false}
        onDeletePoliciesClick={noop}
        currentTeam={{ id: -1, name: "All fleets" }}
        isPremiumTier
        searchQuery=""
        page={0}
        onQueryChange={noop}
        router={mockRouter}
        renderPoliciesCount={() => null}
        count={2}
      />
    );

    expect(screen.getByText("Automations")).toBeInTheDocument();
    expect(screen.getByText("Software, calendar")).toBeInTheDocument();
    expect(screen.getByText("---")).toBeInTheDocument();
  });
});
