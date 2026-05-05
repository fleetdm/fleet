import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";

import WipeHostActivityItem from "./WipedHostActivityItem";

describe("WipeHostActivityItem", () => {
  it("renders the activity content", () => {
    render(
      <WipeHostActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText(/wiped this host/i)).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <WipeHostActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <WipeHostActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
