import { render, screen } from "@testing-library/react";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";

import ResetHostActivityItem from "./ResetHostActivityItem";

describe("ResetHostActivityItem", () => {
  it("renders the activity content", () => {
    render(
      <ResetHostActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText(/reset this host/i)).toBeVisible();
  });
});
