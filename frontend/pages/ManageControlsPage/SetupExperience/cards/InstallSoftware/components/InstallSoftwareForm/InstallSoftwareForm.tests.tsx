import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createMockRouter, createCustomRenderer } from "test/test-utils";
import { noop } from "lodash";
import { SETUP_EXPERIENCE_PLATFORMS } from "interfaces/platform";

import {
  createMockSoftwarePackage,
  createMockSoftwareTitle,
} from "__mocks__/softwareMock";
import mdmAPI from "services/entities/mdm";

import InstallSoftwareForm from "./InstallSoftwareForm";

const render = createCustomRenderer({ withBackendMock: true });

describe("InstallSoftware", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

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

    expect(screen.getByText(/No software available to install/i)).toBeVisible();
    expect(screen.getByRole("button", { name: "Add software" })).toBeVisible();
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

    expect(screen.getByText(/0 software items/)).toBeVisible();
    expect(screen.getByText(/installed during setup/)).toBeVisible();
  });

  it("should render the correct messaging when there are software titles that have been selected to install at setup", async () => {
    const { user } = render(
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

    expect(screen.getByText(/2 software items/)).toBeVisible();
    expect(screen.getByText(/installed during setup/)).toBeVisible();

    await user.hover(screen.getByText("installed during setup"));

    await waitFor(() => {
      const tooltip = screen.getByText(
        /Installation order will depend on software name, starting with 0-9 then A-Z./i
      );
      expect(tooltip).toBeInTheDocument();
    });
  });

  it("should render the correct messaging for Android when there are software titles that have been selected to install at setup", async () => {
    const { user } = render(
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

    expect(screen.getByText(/2 software items/)).toBeVisible();
    expect(screen.getByText(/installed during setup/)).toBeVisible();

    await user.hover(screen.getByText("installed during setup"));

    await waitFor(() => {
      const tooltip = screen.getByText(/Software order will vary/i);
      expect(tooltip).toBeInTheDocument();
    });
  });

  it('should render the "Cancel setup if software fails" form for macos platform', async () => {
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
        name: /Cancel setup if software fails/,
      });
      expect(checkbox).toBeVisible();
      expect(checkbox).toBeChecked();
    });
  });

  it('should render the "Cancel setup if software fails" form for windows platform', async () => {
    render(
      <InstallSoftwareForm
        savedRequireAllSoftwareWindows
        isWindowsMdmEnabled
        currentTeamId={1}
        softwareTitles={[
          createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              install_during_setup: true,
            }),
          }),
          createMockSoftwareTitle(),
        ]}
        hasManualAgentInstall={false}
        platform="windows"
        router={createMockRouter()}
        refetchSoftwareTitles={noop}
      />
    );

    await waitFor(() => {
      const checkbox = screen.getByRole("checkbox", {
        name: /Cancel setup if software fails/,
      });
      expect(checkbox).toBeVisible();
      expect(checkbox).toBeChecked();
    });
  });

  it("calls the Windows require-all API on Save when the Windows checkbox is toggled", async () => {
    const updateRequireAllSoftwareWindowsSpy = jest
      .spyOn(mdmAPI, "updateRequireAllSoftwareWindows")
      .mockResolvedValue({});

    const { user } = render(
      <InstallSoftwareForm
        savedRequireAllSoftwareWindows={false}
        isWindowsMdmEnabled
        currentTeamId={1}
        softwareTitles={[createMockSoftwareTitle()]}
        hasManualAgentInstall={false}
        platform="windows"
        router={createMockRouter()}
        refetchSoftwareTitles={noop}
      />
    );

    await user.click(
      screen.getByRole("checkbox", {
        name: /Cancel setup if software fails/,
      })
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(updateRequireAllSoftwareWindowsSpy).toHaveBeenCalledWith(1, true);
    });
  });

  it("disables the Windows checkbox when Windows MDM is not configured", async () => {
    render(
      <InstallSoftwareForm
        savedRequireAllSoftwareWindows={false}
        isWindowsMdmEnabled={false}
        currentTeamId={1}
        softwareTitles={[createMockSoftwareTitle()]}
        hasManualAgentInstall={false}
        platform="windows"
        router={createMockRouter()}
        refetchSoftwareTitles={noop}
      />
    );

    const checkbox = screen.getByRole("checkbox", {
      name: /Cancel setup if software fails/,
    });
    expect(checkbox).toHaveAttribute("aria-disabled", "true");
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

    const saveButton = screen.getByRole("button", {
      name: "Save",
    });
    expect(saveButton).toBeVisible();
    expect(saveButton).toBeDisabled();
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

      const saveButton = screen.getByRole("button", {
        name: "Save",
      });
      expect(saveButton).toBeVisible();
      expect(saveButton).not.toBeDisabled();
    }
  );
});
