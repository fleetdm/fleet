import React from "react";

import { noop } from "lodash";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import createMockPolicy from "__mocks__/policyMock";

import CalendarEventsModal from "./CalendarEventsModal";

const testGlobalPolicies = [
  createMockPolicy({ team_id: null, name: "Inherited policy 1" }),
  createMockPolicy({ id: 2, team_id: null, name: "Inherited policy 2" }),
  createMockPolicy({ id: 3, team_id: null, name: "Inherited policy 3" }),
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
        policies={testGlobalPolicies}
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
        policies={testGlobalPolicies}
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

  it("disables submission in GitOps mode", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
          // @ts-ignore
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "a.b.cc",
            },
          },
        },
      },
    });

    const onSubmit = jest.fn();

    const { user } = render(
      <CalendarEventsModal
        onExit={noop}
        onSubmit={onSubmit}
        isUpdating={false}
        configured
        enabled
        url="https://server.com/example"
        policies={testGlobalPolicies}
        gitOpsModeEnabled
      />
    );

    expect(screen.queryByText(/Resolution webhook URL/i)).toBeInTheDocument();
    const save = screen.getByRole("button", { name: /Save/i });
    expect(save).toBeInTheDocument();
    await user.click(save);
    expect(onSubmit).not.toHaveBeenCalled();
  });
  it("allows submission", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
        },
      },
    });

    const onSubmit = jest.fn();

    const { user } = render(
      <CalendarEventsModal
        onExit={noop}
        onSubmit={onSubmit}
        isUpdating={false}
        configured
        enabled
        url="https://server.com/example"
        policies={testGlobalPolicies}
      />
    );

    const save = screen.getByRole("button", { name: /Save/i });
    expect(save).toBeInTheDocument();
    await user.click(save);
    expect(onSubmit).toHaveBeenCalled();
  });
});
