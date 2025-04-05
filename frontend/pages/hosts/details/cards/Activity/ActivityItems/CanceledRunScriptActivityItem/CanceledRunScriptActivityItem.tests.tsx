import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import CanceledRunScriptActivityItem from "./CanceledRunScriptActivityItem";

describe("CanceledRunScriptActivityItem", () => {
  const mockActivity = createMockHostPastActivity({
    type: ActivityType.CanceledRunScript,
    details: { script_name: "test.sh" },
  });

  it("renders the activity content", () => {
    render(
      <CanceledRunScriptActivityItem tab="past" activity={mockActivity} />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText(/canceled/i)).toBeVisible();
    expect(screen.getByText(/test.sh/i)).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <CanceledRunScriptActivityItem tab="past" activity={mockActivity} />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <CanceledRunScriptActivityItem tab="past" activity={mockActivity} />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
