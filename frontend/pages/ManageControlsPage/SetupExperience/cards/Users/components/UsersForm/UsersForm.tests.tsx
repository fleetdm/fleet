import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import configAPI from "services/entities/config";
import mdmAPI from "services/entities/mdm";
import teamsAPI from "services/entities/teams";
import { EndUserLocalAccountType } from "interfaces/mdm";

import UsersForm from "./UsersForm";

describe("UsersForm", () => {
  const defaultProps = {
    currentTeamId: 0,
    defaultIsEndUserAuthEnabled: false,
    defaultLockEndUserInfo: false,
    defaultEnableManagedLocalAccount: false,
    defaultEnableManagedLocalAccountWindows: false,
    isIdPConfigured: true,
  };

  const render = createCustomRenderer({
    withBackendMock: true,
  });

  const renderWithMdmEnabled = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: { isMacMdmEnabledAndConfigured: true },
    },
  });

  const renderWithMdmDisabled = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: { isMacMdmEnabledAndConfigured: false },
    },
  });

  const renderWithWindowsMdmEnabled = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isMacMdmEnabledAndConfigured: true,
        isWindowsMdmEnabledAndConfigured: true,
      },
    },
  });

  it("reveals lock end user info when end user auth is toggled on", async () => {
    const { user } = render(<UsersForm {...defaultProps} />);

    expect(screen.queryByText("Lock end user info")).not.toBeInTheDocument();

    await user.click(
      screen.getByRole("checkbox", { name: "Require IdP authentication" })
    );

    expect(screen.getByText("Lock end user info")).toBeInTheDocument();
  });

  it("auto-checks lock end user info on EUA toggle when Apple MDM is configured", async () => {
    const { user } = renderWithMdmEnabled(<UsersForm {...defaultProps} />);

    await user.click(
      screen.getByRole("checkbox", { name: "Require IdP authentication" })
    );

    expect(
      screen.getByRole("checkbox", { name: "Lock end user info" })
    ).toBeChecked();
  });

  it("does not auto-check lock end user info on EUA toggle when Apple MDM is not configured", async () => {
    const { user } = renderWithMdmDisabled(<UsersForm {...defaultProps} />);

    await user.click(
      screen.getByRole("checkbox", { name: "Require IdP authentication" })
    );

    expect(
      screen.getByRole("checkbox", { name: "Lock end user info" })
    ).not.toBeChecked();
  });

  it("preserves the backend lock end user info value on EUA toggle when Apple MDM is not configured", async () => {
    // Simulates: Apple MDM was previously on with lock_end_user_info=true, then
    // Apple MDM was turned off. Toggling EUA must not clobber the saved value.
    const { user } = renderWithMdmDisabled(
      <UsersForm
        {...defaultProps}
        defaultLockEndUserInfo
        defaultIsEndUserAuthEnabled
      />
    );

    const eua = screen.getByRole("checkbox", {
      name: "Require IdP authentication",
    });
    await user.click(eua); // toggle EUA off
    await user.click(eua); // toggle EUA back on

    expect(
      screen.getByRole("checkbox", { name: "Lock end user info" })
    ).toBeChecked();
  });

  it("does not show an 'Advanced options' reveal button", () => {
    render(<UsersForm {...defaultProps} />);
    expect(screen.queryByText("Advanced options")).not.toBeInTheDocument();
  });

  it("renders exactly one Save button", () => {
    render(<UsersForm {...defaultProps} />);
    const saveButtons = screen.getAllByRole("button", { name: "Save" });
    expect(saveButtons).toHaveLength(1);
  });

  describe("save payload", () => {
    afterEach(() => {
      jest.restoreAllMocks();
    });

    it("omits Apple-only fields when Apple MDM is not configured", async () => {
      // The backend skips its EUA->Lock auto-sync when Apple MDM isn't configured, so the
      // FE can safely omit lock_end_user_info; managed local account is rejected outright.
      const updateSpy = jest
        .spyOn(mdmAPI, "updateSetupExperienceSettings")
        .mockResolvedValue({});

      const { user } = renderWithMdmDisabled(<UsersForm {...defaultProps} />);
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(updateSpy).toHaveBeenCalledWith({
        fleet_id: 0,
        enable_end_user_authentication: false,
      });
    });

    it("includes Apple-only fields when Apple MDM is configured", async () => {
      const updateSpy = jest
        .spyOn(mdmAPI, "updateSetupExperienceSettings")
        .mockResolvedValue({});

      const { user } = renderWithMdmEnabled(<UsersForm {...defaultProps} />);
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(updateSpy).toHaveBeenCalledWith({
        fleet_id: 0,
        enable_end_user_authentication: false,
        lock_end_user_info: false,
        enable_managed_local_account: false,
        end_user_local_account_type: EndUserLocalAccountType.ADMIN,
      });
    });
  });

  describe("platform tabs", () => {
    it("hides the Windows tab when Windows MDM is not enabled and configured", () => {
      renderWithMdmEnabled(<UsersForm {...defaultProps} />);
      expect(screen.getByRole("tab", { name: "macOS" })).toBeInTheDocument();
      expect(
        screen.queryByRole("tab", { name: "Windows" })
      ).not.toBeInTheDocument();
    });

    it("shows the Windows tab when Windows MDM is enabled and configured", () => {
      renderWithWindowsMdmEnabled(<UsersForm {...defaultProps} />);
      expect(screen.getByRole("tab", { name: "Windows" })).toBeInTheDocument();
    });

    it("renders the Windows tab without end user account type options", async () => {
      const { user } = renderWithWindowsMdmEnabled(
        <UsersForm {...defaultProps} />
      );
      await user.click(screen.getByRole("tab", { name: "Windows" }));

      expect(
        screen.getByText(
          "End users get the default role for the host's platform.",
          {
            exact: false,
          }
        )
      ).toBeInTheDocument();
      // Account type is macOS-only; Windows gets whatever the platform defaults to.
      expect(
        screen.queryByRole("radio", { name: "Admin" })
      ).not.toBeInTheDocument();
      expect(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      ).toBeInTheDocument();
    });
  });

  describe("windows managed account save", () => {
    it("sends the windows config PATCH on every save, matching the setup experience call", async () => {
      const setupSpy = jest
        .spyOn(mdmAPI, "updateSetupExperienceSettings")
        .mockResolvedValue({});
      const configSpy = jest
        .spyOn(configAPI, "update")
        .mockResolvedValue({} as never);

      const { user } = renderWithWindowsMdmEnabled(
        <UsersForm {...defaultProps} />
      );
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(setupSpy).toHaveBeenCalled();
      // Unconditional: the server only records an activity when the value
      // actually changes, so a no-op PATCH is harmless.
      expect(configSpy).toHaveBeenCalledWith({
        mdm: {
          windows_settings: {
            managed_local_account_settings: { enabled: false },
          },
        },
      });
    });

    it("saves the windows toggle through the config PATCH for no team", async () => {
      jest.spyOn(mdmAPI, "updateSetupExperienceSettings").mockResolvedValue({});
      const configSpy = jest
        .spyOn(configAPI, "update")
        .mockResolvedValue({} as never);

      const { user } = renderWithWindowsMdmEnabled(
        <UsersForm {...defaultProps} />
      );
      await user.click(screen.getByRole("tab", { name: "Windows" }));
      await user.click(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      );
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(configSpy).toHaveBeenCalledWith({
        mdm: {
          windows_settings: {
            managed_local_account_settings: { enabled: true },
          },
        },
      });
    });

    it("saves the windows toggle through the team PATCH for a fleet", async () => {
      jest.spyOn(mdmAPI, "updateSetupExperienceSettings").mockResolvedValue({});
      const teamSpy = jest
        .spyOn(teamsAPI, "updateConfig")
        .mockResolvedValue({} as never);

      const { user } = renderWithWindowsMdmEnabled(
        <UsersForm {...defaultProps} currentTeamId={7} />
      );
      await user.click(screen.getByRole("tab", { name: "Windows" }));
      await user.click(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      );
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(teamSpy).toHaveBeenCalledWith(
        {
          mdm: {
            windows_settings: {
              managed_local_account_settings: { enabled: true },
            },
          },
        },
        7
      );
    });
  });
});
