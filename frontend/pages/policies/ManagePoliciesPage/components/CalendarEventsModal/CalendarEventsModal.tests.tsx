import React from "react";

import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import CalendarEventsModal from "./CalendarEventsModal";

describe("CalendarEventsModal - component", () => {
  it("renders form fields for admin when configured", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
        },
      },
    });

    render(
      <CalendarEventsModal
        configured
        enabled
        url="https://server.com/example"
      />
    );

    expect(screen.queryByText(/Resolution webhook URL/i)).toBeInTheDocument();
  });

  it("hides admin-only fields for team maintainer", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isTeamMaintainer: true,
        },
      },
    });

    render(
      <CalendarEventsModal
        configured
        enabled
        url="https://server.com/example"
      />
    );

    expect(
      screen.queryByText(/Resolution webhook URL/i)
    ).not.toBeInTheDocument();
  });
});
