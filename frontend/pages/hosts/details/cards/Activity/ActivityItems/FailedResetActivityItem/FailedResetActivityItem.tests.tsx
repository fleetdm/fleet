import { render, screen } from "@testing-library/react";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";

import FailedResetActivityItem from "./FailedResetActivityItem";

describe("FailedResetActivityItem", () => {
  it("renders the activity content", () => {
    render(
      <FailedResetActivityItem
        activity={createMockHostPastActivity()}
        tab="past"
      />
    );

    expect(screen.getByText("Reset failed on this host.")).toBeVisible();
  });
});
