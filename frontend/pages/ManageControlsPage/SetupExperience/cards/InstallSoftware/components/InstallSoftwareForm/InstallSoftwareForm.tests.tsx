import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { createMockRouter, renderWithSetup } from "test/test-utils";
import { noop } from "lodash";
import { SETUP_EXPERIENCE_PLATFORMS } from "interfaces/platform";

import {
  createMockSoftwarePackage,
  createMockSoftwareTitle,
} from "__mocks__/softwareMock";

import InstallSoftwareForm from "./InstallSoftwareForm";

describe("InstallSoftware", () => {
  it("should render the expected message if there are no software titles to select from", () => {
    render(
      <InstallSoftwareForm
        savedRequireAllSoftwareMacOS={false}
        currentTeamId={1}
        softwareTitles={null}
        hasManualAgentInstall={false}
        platform="macos"
        router={createMockRouter()}
        refetchSoftwareTitles={noop}
      />
    );

    expect(screen.getByText(/you can add software on the/i)).toBeVisible();
  });

  it("should render the correct messaging when there are software titles but none have been selected to install at setup", () => {
    render(
      <InstallSoftwareForm
        savedRequireAllSoftwareMacOS={false}
        currentTeamId={1}
        softwareTitles={[createMockSoftwareTitle(), createMockSoftwareTitle()]}
        hasManualAgentInstall={false}
        platform="macos"
        router={createMockRouter()}
        refetchSoftwareTitles={noop}
      />
    );

    expect(screen.getByText(/No software selected/)).toBeVisible();
    expect(screen.queryByRole("button", { name: "Add software" })).toBeNull();
  });

  it("should render the correct messaging when there are software titles that have been selected to install at setup", async () => {
    const { user } = renderWithSetup(
      <InstallSoftwareForm
        savedRequireAllSoftwareMacOS={false}
        currentTeamId={1}
        softwareTitles={[
          createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              install_during_setup: true,
            }),
          }),
          createMockSoftwareTitle(
            createMockSoftwareTitle({
              software_package: createMockSoftwarePackage({
                install_during_setup: true,
              }),
            })
          ),
          createMockSoftwareTitle(),
        ]}
        hasManualAgentInstall={false}
        platform="macos"
        router={createMockRouter()}
        refetchSoftwareTitles={noop}
      />
    );

    expect(
      screen.getByText(
        (_, element) =>
          element?.textContent ===
          "2 software items will be installed during setup."
      )
    ).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Select software" })
    ).toBeVisible();

    await user.hover(screen.getByText("installed during setup"));

    await waitFor(() => {
      const tooltip = screen.getByText(
        /Installation order will depend on software name, starting with 0-9 then A-Z./i
      );
      expect(tooltip).toBeInTheDocument();
    });
  });

  it("should render the correct messaging for Android when there are software titles that have been selected to install at setup", async () => {
    const { user } = renderWithSetup(
      <InstallSoftwareForm
        savedRequireAllSoftwareMacOS={false}
        currentTeamId={1}
        softwareTitles={[
          createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              install_during_setup: true,
            }),
          }),
          createMockSoftwareTitle(
            createMockSoftwareTitle({
              software_package: createMockSoftwarePackage({
                install_during_setup: true,
              }),
            })
          ),
          createMockSoftwareTitle(),
        ]}
        hasManualAgentInstall={false}
        platform="android"
        router={createMockRouter()}
        refetchSoftwareTitles={noop}
      />
    );

    expect(
      screen.getByText(
        (_, element) =>
          element?.textContent ===
          "2 software items will be installed during setup."
      )
    ).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Select software" })
    ).toBeVisible();

    await user.hover(screen.getByText("installed during setup"));

    await waitFor(() => {
      const tooltip = screen.getByText(/Software order will vary/i);
      expect(tooltip).toBeInTheDocument();
    });
  });

  it('should render the "Cancel setup if software install fails" form for macos platform', async () => {
    render(
      <InstallSoftwareForm
        savedRequireAllSoftwareMacOS
        currentTeamId={1}
        softwareTitles={[
          createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              install_during_setup: true,
            }),
          }),
          createMockSoftwareTitle(
            createMockSoftwareTitle({
              software_package: createMockSoftwarePackage({
                install_during_setup: true,
              }),
            })
          ),
          createMockSoftwareTitle(),
        ]}
        hasManualAgentInstall={false}
        platform="macos"
        router={createMockRouter()}
        refetchSoftwareTitles={noop}
      />
    );

    await waitFor(() => {
      const checkbox = screen.getByRole("checkbox", {
        name: /Cancel setup if software install fails/,
      });
      expect(checkbox).toBeVisible();
      expect(checkbox).toBeChecked();
    });
  });

  it("should disable adding software for macos with manual agent install", async () => {
    render(
      <InstallSoftwareForm
        savedRequireAllSoftwareMacOS={false}
        currentTeamId={1}
        softwareTitles={[createMockSoftwareTitle(), createMockSoftwareTitle()]}
        hasManualAgentInstall
        platform="macos"
        router={createMockRouter()}
        refetchSoftwareTitles={noop}
      />
    );

    const addSoftwareButton = screen.getByRole("button", {
      name: "Select software",
    });
    expect(addSoftwareButton).toBeVisible();
    expect(addSoftwareButton).toBeDisabled();
  });

  it.each(SETUP_EXPERIENCE_PLATFORMS.filter((val) => val !== "macos"))(
    "should allow adding software for %s platform with manual agent install",
    async (platform) => {
      render(
        <InstallSoftwareForm
          savedRequireAllSoftwareMacOS={false}
          currentTeamId={1}
          softwareTitles={[
            createMockSoftwareTitle(),
            createMockSoftwareTitle(),
          ]}
          hasManualAgentInstall
          platform={platform}
          router={createMockRouter()}
          refetchSoftwareTitles={noop}
        />
      );

      const addSoftwareButton = screen.getByRole("button", {
        name: "Select software",
      });
      expect(addSoftwareButton).toBeVisible();
      expect(addSoftwareButton).not.toBeDisabled();
    }
  );
});
