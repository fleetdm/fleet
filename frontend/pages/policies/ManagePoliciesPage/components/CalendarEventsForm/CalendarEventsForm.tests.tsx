import React from "react";

import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import CalendarEventsForm from "./CalendarEventsForm";

describe("CalendarEventsForm - component", () => {
  it("renders form fields for admin when configured", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
        },
      },
    });

    const { user } = render(
      <CalendarEventsForm configured enabled url="https://server.com/example" />
    );

    expect(screen.queryByText(/Resolution webhook URL/i)).toBeInTheDocument();
    await user.click(
      screen.getByRole("button", { name: /Preview calendar event/i })
    );
    expect(
      screen.getByText(
        /reserved this time to make some changes to your work computer/i
      )
    ).toBeInTheDocument();
  });

  it("hides admin-only fields for team maintainer", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isTeamMaintainer: true,
        },
      },
    });

    const { user } = render(
      <CalendarEventsForm configured enabled url="https://server.com/example" />
    );

    expect(
      screen.queryByText(/Resolution webhook URL/i)
    ).not.toBeInTheDocument();
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
