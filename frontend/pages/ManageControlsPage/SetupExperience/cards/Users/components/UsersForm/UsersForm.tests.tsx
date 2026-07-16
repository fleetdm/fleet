import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import mdmAPI from "services/entities/mdm";
import { EndUserLocalAccountType } from "interfaces/mdm";

import UsersForm from "./UsersForm";

describe("UsersForm", () => {
  const defaultProps = {
    currentTeamId: 0,
    defaultIsEndUserAuthEnabled: false,
    defaultLockEndUserInfo: false,
    defaultEnableManagedLocalAccount: false,
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
});
