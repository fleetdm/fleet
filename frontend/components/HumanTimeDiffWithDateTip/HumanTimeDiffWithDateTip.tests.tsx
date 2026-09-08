import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import {
  HumanTimeDiffWithDateTip,
  HumanTimeDiffWithFleetLaunchCutoff,
} from "./HumanTimeDiffWithDateTip";

const EMPTY_STRING = "Unavailable";
const INVALID_STRING = "Invalid date";
const NEVER_TOOLTIP = "This host has not reported vitals yet.";

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

  it("explains 'Never' on hover when given a never tooltip", async () => {
    const { user } = renderWithSetup(
      <HumanTimeDiffWithFleetLaunchCutoff
        timeString="1970-01-02T00:00:00Z"
        neverTooltip={NEVER_TOOLTIP}
      />
    );

    expect(screen.queryByText(NEVER_TOOLTIP)).not.toBeInTheDocument();

    await user.hover(screen.getByText(/never/i));

    expect(await screen.findByText(NEVER_TOOLTIP)).toBeInTheDocument();
  });

  it("ignores the never tooltip when the date is after Fleet was created", async () => {
    const { user } = renderWithSetup(
      <HumanTimeDiffWithFleetLaunchCutoff
        timeString="2024-04-27T12:00:00Z"
        neverTooltip={NEVER_TOOLTIP}
      />
    );

    await user.hover(screen.getByText(/years ago/i));

    expect(screen.queryByText(NEVER_TOOLTIP)).not.toBeInTheDocument();
  });
});
