import React from "react";

import { noop } from "lodash";
import { fireEvent, screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import createMockPolicy from "__mocks__/policyMock";

import CalendarEventsModal from "./CalendarEventsModal";

const testGlobalPolicy = [
  createMockPolicy({ team_id: null, name: "Inherited policy 1" }),
  createMockPolicy({ id: 2, team_id: null, name: "Inherited policy 2" }),
  createMockPolicy({ id: 3, team_id: null, name: "Inherited policy 3" }),
];

const testTeamPolicies = [
  createMockPolicy({ id: 4, team_id: 2, name: "Team policy 1" }),
  createMockPolicy({ id: 5, team_id: 2, name: "Team policy 2" }),
];

describe("CalendarEventsModal - component", () => {
  it("renders components for admin", async () => {
    const render = createCustomRenderer({
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
        policies={testGlobalPolicy}
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
    const render = createCustomRenderer({
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
        policies={testGlobalPolicy}
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
