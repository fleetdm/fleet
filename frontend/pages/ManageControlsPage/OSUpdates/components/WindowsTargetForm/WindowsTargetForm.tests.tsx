import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { createCustomRenderer } from "test/test-utils";

import WindowsTargetForm from "./WindowsTargetForm";

const renderForm = (isPremiumTier: boolean, currentTeamId = 2) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: { app: { isPremiumTier, config: createMockConfig() } },
  });
  return render(
    <WindowsTargetForm
      currentTeamId={currentTeamId}
      defaultDeadlineDays="5"
      defaultGracePeriodDays="2"
      refetchAppConfig={noop}
      refetchTeamConfig={noop}
    />
  );
};

describe("WindowsTargetForm", () => {
  it("links to the profile builder with Windows and the current fleet on Premium", () => {
    renderForm(true);

    const link = screen.getByRole("link", { name: "profile builder" });
    expect(link).toHaveAttribute(
      "href",
      expect.stringContaining(
        "/controls/os-settings/configuration-profiles/new?fleet_id=2&platform=windows"
      )
    );
  });

  it("keeps the No fleet ID in the link", () => {
    renderForm(true, 0);

    expect(
      screen.getByRole("link", { name: "profile builder" })
    ).toHaveAttribute("href", expect.stringContaining("fleet_id=0"));
  });

  it("does not show the profile builder link on Free", () => {
    renderForm(false);

    expect(
      screen.queryByRole("link", { name: "profile builder" })
    ).not.toBeInTheDocument();
  });
});
