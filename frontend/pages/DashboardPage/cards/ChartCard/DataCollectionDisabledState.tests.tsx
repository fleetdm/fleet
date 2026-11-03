import { screen } from "@testing-library/react";
import React from "react";

import createMockUser from "__mocks__/userMock";
import { createCustomRenderer } from "test/test-utils";

import DataCollectionDisabledState from "./DataCollectionDisabledState";

describe("DataCollectionDisabledState", () => {
  it("shows Turn on for a global admin", () => {
    const render = createCustomRenderer({
      context: { app: { isGlobalAdmin: true } },
    });
    render(
      <DataCollectionDisabledState
        datasetLabel="Hosts online"
        globallyEnabled
        currentTeamId={5}
      />
    );

    expect(
      screen.getByRole("button", { name: /Turn on/i })
    ).toBeInTheDocument();
  });

  it("shows Turn on when the viewer is a team admin of the passed-in fleet, even if the app context's selected team is different", () => {
    const teamAdminOfFleet5 = createMockUser({
      global_role: null,
      teams: [{ id: 5, name: "Fleet 5", role: "admin" }],
    });
    const render = createCustomRenderer({
      context: {
        app: {
          currentUser: teamAdminOfFleet5,
          isGlobalAdmin: false,
          // Reflects a different fleet selected in the app (or none), which is
          // what happens when the host page is opened by URL directly.
          isTeamAdmin: false,
        },
      },
    });
    render(
      <DataCollectionDisabledState
        datasetLabel="Hosts online"
        globallyEnabled
        currentTeamId={5}
      />
    );

    expect(
      screen.getByRole("button", { name: /Turn on/i })
    ).toBeInTheDocument();
    expect(screen.queryByText(/Ask an admin/i)).not.toBeInTheDocument();
  });

  it("shows Ask an admin when the viewer is not admin of the passed-in fleet", () => {
    const teamObserverOfFleet5 = createMockUser({
      global_role: null,
      teams: [{ id: 5, name: "Fleet 5", role: "observer" }],
    });
    const render = createCustomRenderer({
      context: {
        app: {
          currentUser: teamObserverOfFleet5,
          isGlobalAdmin: false,
          isTeamAdmin: false,
        },
      },
    });
    render(
      <DataCollectionDisabledState
        datasetLabel="Hosts online"
        globallyEnabled
        currentTeamId={5}
      />
    );

    expect(screen.getByText(/Ask an admin to turn on/i)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Turn on/i })
    ).not.toBeInTheDocument();
  });

  it("hides Turn on from a team admin when the setting is disabled globally — only a global admin can override the org gate", () => {
    const teamAdminOfFleet5 = createMockUser({
      global_role: null,
      teams: [{ id: 5, name: "Fleet 5", role: "admin" }],
    });
    const render = createCustomRenderer({
      context: {
        app: {
          currentUser: teamAdminOfFleet5,
          isGlobalAdmin: false,
        },
      },
    });
    render(
      <DataCollectionDisabledState
        datasetLabel="Hosts online"
        globallyEnabled={false}
        currentTeamId={5}
      />
    );

    expect(screen.getByText(/Ask an admin to turn on/i)).toBeInTheDocument();
    // Scope tracks the user's fleet context, not where the fix lives — a
    // team-admin viewing their fleet's dashboard shouldn't see "all fleets".
    expect(screen.getByText(/for this fleet/i)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Turn on/i })
    ).not.toBeInTheDocument();
  });

  it("shows Turn on to a global admin when the setting is disabled globally, scope stays on the current fleet", () => {
    const render = createCustomRenderer({
      context: { app: { isGlobalAdmin: true } },
    });
    render(
      <DataCollectionDisabledState
        datasetLabel="Hosts online"
        globallyEnabled={false}
        currentTeamId={5}
      />
    );

    expect(
      screen.getByRole("button", { name: /Turn on/i })
    ).toBeInTheDocument();
    expect(screen.getByText(/for this fleet/i)).toBeInTheDocument();
  });
});
