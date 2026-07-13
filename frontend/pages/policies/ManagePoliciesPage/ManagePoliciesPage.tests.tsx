import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import {
  createCustomRenderer,
  createMockRouter,
  baseUrl,
} from "test/test-utils";
import mockServer from "test/mock-server";
import createMockConfig from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";
import createMockPolicy from "__mocks__/policyMock";

import ManagePoliciesPage from "./ManagePoliciesPage";

const FLEET_WITH_POLICIES_ID = 1;
const FLEET_WITHOUT_POLICIES_ID = 2;

const getConfigHandler = () =>
  http.get(baseUrl("/config"), () => HttpResponse.json(createMockConfig()));

const getFleetHandler = () =>
  http.get(baseUrl("/fleets/:id"), ({ params }) => {
    const fleetId = Number(params.id);
    const fleet = {
      ...createMockConfig(),
      id: fleetId,
      name: `Fleet ${fleetId}`,
    };
    return HttpResponse.json({ team: fleet, fleet });
  });

const getFleetPoliciesHandler = () =>
  http.get(baseUrl("/fleets/:fleetId/policies"), ({ params }) => {
    const fleetId = Number(params.fleetId);
    const policies =
      fleetId === FLEET_WITH_POLICIES_ID
        ? [
            createMockPolicy({
              id: 1,
              name: "Software policy",
              team_id: fleetId,
              install_software: { name: "Zoom", software_title_id: 1 },
            }),
          ]
        : [];
    return HttpResponse.json({ policies });
  });

const getFleetPoliciesCountHandler = () =>
  http.get(baseUrl("/fleets/:fleetId/policies/count"), ({ params }) => {
    const fleetId = Number(params.fleetId);
    return HttpResponse.json({
      count: fleetId === FLEET_WITH_POLICIES_ID ? 1 : 0,
    });
  });

const getGlobalPoliciesHandler = () =>
  http.get(baseUrl("/policies"), () => HttpResponse.json({ policies: [] }));

const getGlobalPoliciesCountHandler = () =>
  http.get(baseUrl("/policies/count"), () => HttpResponse.json({ count: 0 }));

const getAutomationFilterControl = (): HTMLElement => {
  const control = document.querySelector(
    ".manage-policies-page__filter-automation-dropdown .react-select__control"
  );
  if (!control) {
    throw new Error("Automations filter control not found");
  }
  return control as HTMLElement;
};

const setupHandlers = () => {
  mockServer.use(
    getConfigHandler(),
    getFleetHandler(),
    getFleetPoliciesHandler(),
    getFleetPoliciesCountHandler(),
    getGlobalPoliciesHandler(),
    getGlobalPoliciesCountHandler()
  );
};

describe("ManagePoliciesPage - automations filter", () => {
  const renderPage = (fleetId: string, automationType?: string) => {
    setupHandlers();

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser({ global_role: "admin" }),
          isGlobalAdmin: true,
          isOnGlobalTeam: true,
          isPremiumTier: true,
          availableTeams: [
            { id: -1, name: "All fleets" },
            { id: 0, name: "Unassigned" },
            { id: FLEET_WITH_POLICIES_ID, name: "Fleet with policies" },
            { id: FLEET_WITHOUT_POLICIES_ID, name: "Fleet without policies" },
          ],
          config: createMockConfig(),
          setCurrentTeam: jest.fn(),
          setFilteredPoliciesPath: jest.fn(),
          setConfig: jest.fn(),
        },
      },
    });

    const query: Record<string, string> = { fleet_id: fleetId };
    if (automationType) {
      query.automation_type = automationType;
    }

    return render(
      <ManagePoliciesPage
        router={createMockRouter()}
        location={{
          action: "PUSH",
          hash: "",
          key: "",
          pathname: "/policies/manage",
          query,
          search: `?${new URLSearchParams(query).toString()}`,
        }}
      />
    );
  };

  it("keeps the automations filter dropdown visible and enabled for a fleet with no policies when a filter is present in the URL", async () => {
    renderPage(FLEET_WITHOUT_POLICIES_ID.toString(), "software");

    await waitFor(() => {
      expect(
        screen.getByText("No policies match the current filters.")
      ).toBeInTheDocument();
    });

    // Per the fix for #44624: when a filter is present in the query params,
    // the automations filter dropdown must remain visible and enabled (and
    // reflect the selected filter) even though this fleet has zero policies.
    expect(screen.getByText("Software")).toBeInTheDocument();
    expect(getAutomationFilterControl()).not.toHaveClass(
      "react-select__control--is-disabled"
    );
  });

  it("offers software/scripts/conditional access automation types for the 'Unassigned' fleet, but not calendar", async () => {
    const { user } = renderPage("0", "software");

    await waitFor(() => {
      expect(
        screen.getByText("No policies match the current filters.")
      ).toBeInTheDocument();
    });

    // The selected filter persists (sticky), matching what's in the URL.
    expect(screen.getByText("Software")).toBeInTheDocument();

    // "Unassigned" is a real, policy-bearing fleet (unlike "All fleets", which
    // is restricted to webhook/ticket-only automations) -- it should offer
    // every automation type EXCEPT calendar events, which
    // PolicyAutomationsFields hardcodes as fleet-only (never available for
    // "All fleets" or "Unassigned").
    await user.click(getAutomationFilterControl());

    expect(screen.getByText("Scripts")).toBeInTheDocument();
    expect(screen.getByText("Conditional access")).toBeInTheDocument();
    expect(screen.queryByText("Calendar")).not.toBeInTheDocument();
  });

  it("rejects an automation_type=calendar query param for the 'Unassigned' fleet, since calendar isn't a valid option there", async () => {
    renderPage("0", "calendar");

    await waitFor(() => {
      expect(
        screen.getByText("No policies for this fleet")
      ).toBeInTheDocument();
    });

    // "calendar" isn't a valid filter for "Unassigned", so it must not stick
    // as the selected value -- the filter should fall back to its default.
    expect(screen.getByText("All automations")).toBeInTheDocument();
    expect(screen.queryByText("Calendar")).not.toBeInTheDocument();
  });
});
