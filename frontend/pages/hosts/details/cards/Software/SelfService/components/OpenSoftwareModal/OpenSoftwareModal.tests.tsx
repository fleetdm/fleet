import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import { noop } from "lodash";
import OpenSoftwareModal from "./OpenSoftwareModal";

describe("OpenSoftwareModal", () => {
  it("renders macOS instructions when softwareSource is 'apps'", () => {
    render(
      <OpenSoftwareModal
        softwareSource="apps"
        softwareName="Slack"
        onExit={noop}
      />
    );

    expect(screen.getAllByText(/Slack/i)).toHaveLength(2); // Says Slack twice
    expect(screen.getByText(/Finder > Applications/i)).toBeVisible();
    expect(screen.getByText(/and double-click it/i)).toBeVisible();

    expect(screen.getByRole("button", { name: /Done/i })).toBeVisible();
  });

  it("renders Windows instructions when softwareSource is 'programs'", () => {
    render(
      <OpenSoftwareModal
        softwareSource="programs"
        softwareName="Zoom"
        onExit={noop}
      />
    );

    expect(screen.getByText(/Find/i)).toBeVisible();
    expect(screen.getByText(/Zoom/i)).toBeVisible();
    expect(screen.getByText(/Start Menu/i)).toBeVisible();
    expect(screen.getByText(/and click it/i)).toBeVisible();

    expect(screen.getByRole("button", { name: /Done/i })).toBeVisible();
  });

  it("renders nothing when softwareSource is unknown", () => {
    render(
      <OpenSoftwareModal
        softwareSource="chrome_extensions"
        softwareName="Chrome extension"
        onExit={noop}
      />
    );

    // should not find expected text
    expect(
      screen.queryByText(/Find Chrome extension/i)
    ).not.toBeInTheDocument();
  });

  it("calls onExit when Done button is clicked", async () => {
    const onExitMock = jest.fn();
    const { user } = renderWithSetup(
      <OpenSoftwareModal
        softwareSource="apps"
        softwareName="Slack"
        onExit={onExitMock}
      />
    );

    await user.click(screen.getByRole("button", { name: /Done/i }));
    expect(onExitMock).toHaveBeenCalled();
  });
});
