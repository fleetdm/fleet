import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import {
  HumanTimeDiffWithDateTip,
  HumanTimeDiffWithFleetLaunchCutoff,
} from "./HumanTimeDiffWithDateTip";

const EMPTY_STRING = "Unavailable";
const INVALID_STRING = "Invalid date";

describe("HumanTimeDiffWithDateTip - component", () => {
  it("renders tooltip on hover", async () => {
    const { user } = renderWithSetup(
      <HumanTimeDiffWithDateTip timeString="2015-12-06T10:30:00Z" />
    );

    // Note: number of years varies over time
    await user.hover(screen.getByText(/years ago/i));

    // Note: hour of day varies for timezones
    expect(screen.getByText(/12\/6\/2015/i)).toBeInTheDocument();
  });

  it("handles empty string error", async () => {
    render(<HumanTimeDiffWithDateTip timeString="" />);

    const emptyStringText = screen.getByText(EMPTY_STRING);
    expect(emptyStringText).toBeInTheDocument();
  });

  it("handles invalid string error", async () => {
    render(<HumanTimeDiffWithDateTip timeString="foobar" />);

    const invalidStringText = screen.getByText(INVALID_STRING);
    expect(invalidStringText).toBeInTheDocument();
  });

  it("returns never if configured to cutoff dates before Fleet was created", async () => {
    render(
      <HumanTimeDiffWithFleetLaunchCutoff timeString="1970-01-02T00:00:00Z" />
    );

    expect(screen.getByText(/never/i)).toBeInTheDocument();
  });
});
