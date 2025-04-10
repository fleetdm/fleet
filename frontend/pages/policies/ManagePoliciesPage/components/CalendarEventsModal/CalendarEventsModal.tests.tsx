import React from "react";

import { http, HttpResponse } from "msw";
import { noop } from "lodash";
import { fireEvent, screen } from "@testing-library/react";
import { baseUrl, createCustomRenderer } from "test/test-utils";
import createMockPolicy from "__mocks__/policyMock";
import mockServer from "test/mock-server";

import CalendarEventsModal from "./CalendarEventsModal";

const globalPoliciesHandler = http.get(baseUrl("/policies"), () => {
  return HttpResponse.json({
    policies: [
      createMockPolicy({ team_id: null, name: "Inherited policy 1" }),
      createMockPolicy({ id: 2, team_id: null, name: "Inherited policy 2" }),
      createMockPolicy({ id: 3, team_id: null, name: "Inherited policy 3" }),
    ],
  });
});

const teamPoliciesHandler = http.get(baseUrl("/teams/2/policies"), () => {
  return HttpResponse.json({
    policies: [
      createMockPolicy({ id: 4, team_id: 2, name: "Team policy 1" }),
      createMockPolicy({ id: 5, team_id: 2, name: "Team policy 2" }),
    ],
  });
});

describe("CalendarEventsModal - component", () => {
  it("renders components for admin", async () => {
    mockServer.use(globalPoliciesHandler);
    mockServer.use(teamPoliciesHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
        },
      },
    });

    const { user } = render(
      <CalendarEventsModal
        onExit={noop}
        onSubmit={noop}
        isUpdating={false}
        configured
        enabled
        url="https://server.com/example"
        teamId={2}
      />
    );

    expect(screen.queryByText(/Resolution webhook URL/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Save/i })).toBeInTheDocument();
    await user.click(
      screen.getByRole("button", { name: /Preview calendar event/i })
    );
    expect(
      screen.getByText(
        /reserved this time to make some changes to your work computer/i
      )
    ).toBeInTheDocument();
  });

  it("renders limited components for team maintainer", async () => {
    mockServer.use(globalPoliciesHandler);
    mockServer.use(teamPoliciesHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isTeamMaintainer: true,
        },
      },
    });

    const { user } = render(
      <CalendarEventsModal
        onExit={noop}
        onSubmit={noop}
        isUpdating={false}
        configured
        enabled
        url="https://server.com/example"
        teamId={2}
      />
    );

    expect(screen.queryByText(/enabled/i)).not.toBeInTheDocument(); // Admin only
    expect(
      screen.queryByText(/Resolution webhook URL/i)
    ).not.toBeInTheDocument(); // Admin only
    expect(screen.queryByRole("button", { name: /Save/i })).toBeInTheDocument();
    await user.click(
      screen.getByRole("button", { name: /Preview calendar event/i })
    );
    expect(
      screen.getByText(
        /reserved this time to make some changes to your work computer/i
      )
    ).toBeInTheDocument();
  });
});
