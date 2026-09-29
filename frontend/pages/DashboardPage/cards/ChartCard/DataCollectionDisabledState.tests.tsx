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
        currentTeamId={5}
      />
    );

    expect(screen.getByText(/Ask an admin to turn on/i)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Turn on/i })
    ).not.toBeInTheDocument();
  });
});
