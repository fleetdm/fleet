import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import UninstallSoftwareModal from "./UninstallSoftwareModal";

describe("UninstallSoftwareModal", () => {
  it("renders the generic uninstall message with software name", () => {
    render(
      <UninstallSoftwareModal
        softwareId={1}
        softwareName="Slack"
        token="abc"
        onExit={noop}
        onSuccess={noop}
      />
    );

    expect(
      screen.getByText(
        /Uninstalling this software will remove it and may remove Slack data from your device/i
      )
    ).toBeVisible();
    expect(screen.getByRole("button", { name: /Uninstall/i })).toBeVisible();
    expect(screen.getByRole("button", { name: /Cancel/i })).toBeVisible();
  });

  it("renders the generic uninstall message with default software name", () => {
    render(
      <UninstallSoftwareModal
        softwareId={1}
        token="abc"
        onExit={noop}
        onSuccess={noop}
      />
    );

    expect(
      screen.getByText(/Uninstalling this software will remove it/i)
    ).toBeVisible();
  });
});
