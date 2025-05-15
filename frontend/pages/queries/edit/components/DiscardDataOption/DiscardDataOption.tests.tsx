import React from "react";

import { fireEvent, render, screen } from "@testing-library/react";

import DiscardDataOption from "./DiscardDataOption";

describe("DiscardDataOption component", () => {
  const selectedLoggingType = "snapshot";
  const [discardData, setDiscardData] = [false, jest.fn()];

  it("Renders normal help text when the global option is not disabled", () => {
    render(
      <DiscardDataOption
        queryReportsDisabled={false}
        {...{ selectedLoggingType, discardData, setDiscardData }}
      />
    );

    expect(screen.getByText(/Discard data/)).toBeInTheDocument();
    expect(
      screen.getByText(
        /The most recent results for each host will not be available in Fleet./
      )
    ).toBeInTheDocument();
  });

  it('Renders the "disabled" help text with tooltip when the global option is disabled', async () => {
    render(
      <DiscardDataOption
        queryReportsDisabled
        {...{ selectedLoggingType, discardData, setDiscardData }}
      />
    );

    expect(screen.getByText(/Discard data/)).toBeInTheDocument();
    expect(screen.getByText(/This setting is ignored/)).toBeInTheDocument();
  });

  it('Restores normal help text when disabled and then "Edit anyway" is clicked', async () => {
    render(
      <DiscardDataOption
        queryReportsDisabled
        {...{ selectedLoggingType, discardData, setDiscardData }}
      />
    );

    // disabled
    expect(screen.getByText(/Discard data/)).toBeInTheDocument();
    expect(screen.getByText(/This setting is ignored/)).toBeInTheDocument();

    // enable
    await fireEvent.click(screen.getByText(/Edit anyway/));

    // normal text
    expect(
      screen.getByText(
        /The most recent results for each host will not be available in Fleet./
      )
    ).toBeInTheDocument();
  });
  it('Renders the info banner when  "Differential"  logging option is selected', () => {
    render(
      <DiscardDataOption
        selectedLoggingType="differential"
        queryReportsDisabled={false}
        {...{ discardData, setDiscardData }}
      />
    );

    expect(
      screen.getByText(
        /setting is ignored when differential logging is enabled. This/
      )
    ).toBeInTheDocument();
  });
  it('Renders the info banner when  "Differential (ignore removals)" logging option is selected', () => {
    render(
      <DiscardDataOption
        selectedLoggingType="differential_ignore_removals"
        queryReportsDisabled={false}
        {...{ discardData, setDiscardData }}
      />
    );
    expect(
      screen.getByText(
        /setting is ignored when differential logging is enabled. This/
      )
    ).toBeInTheDocument();
  });
});
