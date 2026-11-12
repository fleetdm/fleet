import { screen } from "@testing-library/react";
import React from "react";
import { QueryClient } from "react-query";

import { notify } from "components/ToastNotification";
import { EndUserLocalAccountType } from "interfaces/mdm";
import configAPI from "services/entities/config";
import mdmAPI from "services/entities/mdm";
import teamsAPI from "services/entities/teams";
import { createCustomRenderer } from "test/test-utils";

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
        macos_managed_local_account_enabled: false,
        macos_end_user_local_account_type: EndUserLocalAccountType.ADMIN,
      });
    });
  });

  describe("platform tabs", () => {
    it("keeps the Windows tab but disables the checkbox when Windows MDM is not configured", async () => {
      const { user } = renderWithMdmEnabled(<UsersForm {...defaultProps} />);
      expect(screen.getByRole("tab", { name: "macOS" })).toBeInTheDocument();

      await user.click(screen.getByRole("tab", { name: "Windows" }));

      // The Checkbox renders a div with role="checkbox", so aria-disabled is the disabled signal.
      expect(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      ).toHaveAttribute("aria-disabled", "true");
    });

    it("enables the checkbox when Windows MDM is enabled and configured", async () => {
      const { user } = renderWithWindowsMdmEnabled(
        <UsersForm {...defaultProps} />
      );
      await user.click(screen.getByRole("tab", { name: "Windows" }));

      expect(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      ).toHaveAttribute("aria-disabled", "false");
    });

    // Which section each tab renders. The sections' own contents are their components' concern, so this only pins the wiring.
    it("swaps the macOS section for the Windows one", async () => {
      const { user } = renderWithWindowsMdmEnabled(
        <UsersForm {...defaultProps} />
      );
      expect(screen.getByRole("radio", { name: "Admin" })).toBeInTheDocument();

      await user.click(screen.getByRole("tab", { name: "Windows" }));

      expect(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      ).toBeInTheDocument();
    });
  });

  describe("windows managed account save", () => {
    afterEach(() => {
      jest.restoreAllMocks();
    });

    const renderWithOnlyWindowsMdm = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isMacMdmEnabledAndConfigured: false,
          isWindowsMdmEnabledAndConfigured: true,
        },
      },
    });

    const spyOnSaveCalls = () => ({
      setup: jest
        .spyOn(mdmAPI, "updateSetupExperienceSettings")
        .mockResolvedValue({}),
      config: jest.spyOn(configAPI, "update").mockResolvedValue({} as never),
      team: jest.spyOn(teamsAPI, "updateConfig").mockResolvedValue({} as never),
    });

    // Both platforms go in one setup experience request, never through the
    // config or fleet APIs. The untouched-toggle row also pins that the value
    // is sent on every save, not only when it changed.
    it.each([
      { target: "unassigned", currentTeamId: 0, turnOn: false },
      { target: "unassigned", currentTeamId: 0, turnOn: true },
      { target: "a fleet", currentTeamId: 7, turnOn: true },
    ])(
      "saves both platforms in one request for $target (turn on: $turnOn)",
      async ({ currentTeamId, turnOn }) => {
        const spies = spyOnSaveCalls();
        const { user } = renderWithWindowsMdmEnabled(
          <UsersForm {...defaultProps} currentTeamId={currentTeamId} />
        );
        if (turnOn) {
          await user.click(screen.getByRole("tab", { name: "Windows" }));
          await user.click(
            screen.getByRole("checkbox", { name: "Create hidden admin" })
          );
        }
        await user.click(screen.getByRole("button", { name: "Save" }));

        expect(spies.setup).toHaveBeenCalledTimes(1);
        expect(spies.setup).toHaveBeenCalledWith({
          fleet_id: currentTeamId,
          enable_end_user_authentication: false,
          lock_end_user_info: false,
          macos_managed_local_account_enabled: false,
          macos_end_user_local_account_type: EndUserLocalAccountType.ADMIN,
          windows_managed_local_account_enabled: turnOn,
        });
        expect(spies.config).not.toHaveBeenCalled();
        expect(spies.team).not.toHaveBeenCalled();
      }
    );

    it("sends only the Windows setting when Apple MDM is off", async () => {
      const spies = spyOnSaveCalls();
      const { user } = renderWithOnlyWindowsMdm(
        <UsersForm {...defaultProps} defaultEnableManagedLocalAccountWindows />
      );
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(spies.setup).toHaveBeenCalledWith({
        fleet_id: 0,
        enable_end_user_authentication: false,
        windows_managed_local_account_enabled: true,
      });
    });

    it("leaves out the Windows setting when Windows MDM is off", async () => {
      const spies = spyOnSaveCalls();
      const { user } = renderWithMdmEnabled(
        <UsersForm {...defaultProps} currentTeamId={7} />
      );
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(spies.setup).toHaveBeenCalledTimes(1);
      expect(spies.setup.mock.calls[0][0]).not.toHaveProperty(
        "windows_managed_local_account_enabled"
      );
    });

    // Several other cards read the app config and the fleet from these same
    // cache keys, so a save has to drop them. The fleet key is only
    // invalidated when there is a fleet: unassigned has no such query.
    it.each([
      { target: "unassigned", currentTeamId: 0, expectsTeamKey: false },
      { target: "a fleet", currentTeamId: 7, expectsTeamKey: true },
    ])(
      "invalidates the cached config for $target after saving",
      async ({ currentTeamId, expectsTeamKey }) => {
        spyOnSaveCalls();
        const invalidate = jest.spyOn(
          QueryClient.prototype,
          "invalidateQueries"
        );

        const { user } = renderWithWindowsMdmEnabled(
          <UsersForm {...defaultProps} currentTeamId={currentTeamId} />
        );
        await user.click(screen.getByRole("button", { name: "Save" }));

        expect(invalidate).toHaveBeenCalledWith(["config"]);
        if (expectsTeamKey) {
          expect(invalidate).toHaveBeenCalledWith(["team", currentTeamId]);
        } else {
          expect(invalidate).not.toHaveBeenCalledWith(["team", currentTeamId]);
        }
      }
    );

    it("shows the error message when the save fails", async () => {
      const err = new Error("missing or invalid license");
      jest
        .spyOn(mdmAPI, "updateSetupExperienceSettings")
        .mockRejectedValue(err);
      const notifyError = jest.spyOn(notify, "error");
      const { user } = renderWithWindowsMdmEnabled(
        <UsersForm {...defaultProps} />
      );
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(notifyError).toHaveBeenCalledWith(
        "Couldn't update settings. Please try again.",
        {
          response: err,
        }
      );
    });
  });
});
