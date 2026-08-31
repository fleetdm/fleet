import React from "react";
import { QueryClient } from "react-query";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import configAPI from "services/entities/config";
import mdmAPI from "services/entities/mdm";
import teamsAPI from "services/entities/teams";
import { EndUserLocalAccountType } from "interfaces/mdm";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

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

    const windowsPayload = (enabled: boolean) => ({
      mdm: {
        windows_settings: { enable_managed_local_account: enabled },
      },
    });

    const spyOnSaveCalls = () => ({
      setup: jest
        .spyOn(mdmAPI, "updateSetupExperienceSettings")
        .mockResolvedValue({}),
      config: jest.spyOn(configAPI, "update").mockResolvedValue({} as never),
      team: jest.spyOn(teamsAPI, "updateConfig").mockResolvedValue({} as never),
    });

    // No team saves through the config API, a fleet through the team API. The untouched-toggle row also pins that the
    // value is sent on every save, not only when it changed.
    it.each([
      { target: "no team", currentTeamId: 0, turnOn: false, enabled: false },
      { target: "no team", currentTeamId: 0, turnOn: true, enabled: true },
      { target: "a fleet", currentTeamId: 7, turnOn: true, enabled: true },
    ])(
      "saves the windows toggle as $enabled for $target",
      async ({ currentTeamId, turnOn, enabled }) => {
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

        expect(spies.setup).toHaveBeenCalled();
        // Asserting the unused API was NOT called
        if (currentTeamId === APP_CONTEXT_NO_TEAM_ID) {
          expect(spies.config).toHaveBeenCalledWith(windowsPayload(enabled));
          expect(spies.team).not.toHaveBeenCalled();
        } else {
          expect(spies.team).toHaveBeenCalledWith(
            windowsPayload(enabled),
            currentTeamId
          );
          expect(spies.config).not.toHaveBeenCalled();
        }
      }
    );

    // Windows MDM off means the tab is never rendered, so there is nothing the user could have changed. Both branches
    // are covered because each would otherwise send through a different API.
    it.each([
      { target: "no team", currentTeamId: 0 },
      { target: "a fleet", currentTeamId: 7 },
    ])(
      "skips the windows PATCH for $target when windows mdm is not configured",
      async ({ currentTeamId }) => {
        const spies = spyOnSaveCalls();
        const { user } = render(
          <UsersForm {...defaultProps} currentTeamId={currentTeamId} />
        );
        await user.click(screen.getByRole("button", { name: "Save" }));

        // The rest of the form still saves; only the Windows call is skipped.
        expect(spies.setup).toHaveBeenCalled();
        expect(spies.config).not.toHaveBeenCalled();
        expect(spies.team).not.toHaveBeenCalled();
      }
    );

    // Several other cards read the app config and the fleet from these same cache keys, so a save has to drop them.
    // The fleet key is only invalidated when there is a fleet: no-team has no such query to begin with.
    it.each([
      { target: "no team", currentTeamId: 0, expectsTeamKey: false },
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
  });
});
