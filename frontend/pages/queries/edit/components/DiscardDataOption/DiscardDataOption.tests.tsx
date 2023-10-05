import React from "react";

import { fireEvent, render, screen } from "@testing-library/react";

import createMockConfig from "__mocks__/configMock";
import DiscardDataOption from "./DiscardDataOption";

describe("DiscardDataOption component", () => {
  const selectedLoggingType = "snapshot";
  const [discardData, setDiscardData] = [false, jest.fn()];

  it("Renders normal help text when the global option is not disabled", () => {
    const appConfig = createMockConfig();
    render(
      <DiscardDataOption
        {...{ appConfig, selectedLoggingType, discardData, setDiscardData }}
      />
    );

    expect(screen.getByText(/Discard data/)).toBeInTheDocument();
    expect(screen.getByText(/Data will still be sent/)).toBeInTheDocument();
  });

  it('Renders the "disabled" help text with tooltip when the global option is disabled', async () => {
    const appConfig = createMockConfig({
      server_settings: {
        query_reports_disabled: true,
        // below fields are not used in this test
        server_url: "https://localhost:8080",
        live_query_disabled: false,
        enable_analytics: true,
        deferred_save_host: false,
      },
    });
    render(
      <DiscardDataOption
        {...{ appConfig, selectedLoggingType, discardData, setDiscardData }}
      />
    );

    expect(screen.getByText(/Discard data/)).toBeInTheDocument();
    expect(screen.getByText(/This setting is ignored/)).toBeInTheDocument();

    await fireEvent.mouseOver(screen.getByText(/globally disabled/));

    expect(screen.getByText(/A Fleet administrator/)).toBeInTheDocument();
  });

  it('Restores normal help text when disabled and then "Edit anyway" is clicked', async () => {
    const appConfig = createMockConfig({
      server_settings: {
        query_reports_disabled: true,
        // below fields are not used in this test
        server_url: "https://localhost:8080",
        live_query_disabled: false,
        enable_analytics: true,
        deferred_save_host: false,
      },
    });
    render(
      <DiscardDataOption
        {...{ appConfig, selectedLoggingType, discardData, setDiscardData }}
      />
    );

    // disabled
    expect(screen.getByText(/Discard data/)).toBeInTheDocument();
    expect(screen.getByText(/This setting is ignored/)).toBeInTheDocument();

    // enable
    await fireEvent.click(screen.getByText(/Edit anyway/));

    // normal text
    expect(screen.getByText(/Data will still be sent/)).toBeInTheDocument();
  });
  it('Renders the info banner when  "Differential"  logging option is selected', () => {
    const appConfig = createMockConfig({});
    render(
      <DiscardDataOption
        {...{ appConfig, discardData, setDiscardData }}
        selectedLoggingType="differential"
      />
    );

    expect(
      screen.getByText(
        /setting is ignored when differential logging is enabled. This/
      )
    ).toBeInTheDocument();
  });
  it('Renders the info banner when  "Differential (ignore removals)" logging option is selected', () => {
    const appConfig = createMockConfig({});
    render(
      <DiscardDataOption
        {...{ appConfig, discardData, setDiscardData }}
        selectedLoggingType="differential_ignore_removals"
      />
    );
    expect(
      screen.getByText(
        /setting is ignored when differential logging is enabled. This/
      )
    ).toBeInTheDocument();
  });
});
