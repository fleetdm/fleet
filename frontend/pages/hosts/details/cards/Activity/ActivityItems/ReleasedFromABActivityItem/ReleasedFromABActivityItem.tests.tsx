import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";
import ReleasedFromABActivityItem from "./ReleasedFromABActivityItem";

describe("ReleasedFromABActivityItem", () => {
  const mockActivity = createMockHostPastActivity({
    type: ActivityType.ReleasedDeviceFromAB,
    details: {},
  });

  it("renders the activity content", () => {
    render(<ReleasedFromABActivityItem tab="past" activity={mockActivity} />);

    expect(screen.getByText("Test User")).toBeVisible();
    expect(
      screen.getByText(/released this host from Apple Business/i)
    ).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(<ReleasedFromABActivityItem tab="past" activity={mockActivity} />);

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(<ReleasedFromABActivityItem tab="past" activity={mockActivity} />);

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
