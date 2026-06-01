import React from "react";

import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import CalendarEventsModal from "./CalendarEventsModal";

describe("CalendarEventsModal - component", () => {
  it("renders form fields when configured", () => {
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

  it("renders the not-configured placeholder copy for a global admin with a link to Settings", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
        },
      },
    });

    render(<CalendarEventsModal configured={false} enabled={false} url="" />);

    expect(
      screen.getByRole("link", { name: /Settings.*Integrations.*Calendars/i })
    ).toBeInTheDocument();
  });

  it("renders the not-configured placeholder copy without a link for a team admin", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isTeamAdmin: true,
        },
      },
    });

    render(<CalendarEventsModal configured={false} enabled={false} url="" />);

    expect(
      screen.queryByRole("link", {
        name: /Settings.*Integrations.*Calendars/i,
      })
    ).not.toBeInTheDocument();
    expect(screen.getByText(/Settings/i)).toBeInTheDocument();
  });
});
