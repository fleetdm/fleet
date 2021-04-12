import helpers from "components/UserRow/helpers";
import { userStub } from "test/stubs";

describe("UserRow - helpers", () => {
  describe("#userActionOptions", () => {
    const { userActionOptions } = helpers;

    it("returns the correct options for invites", () => {
      expect(userActionOptions(false, userStub, true)).toEqual([
        {
          disabled: false,
          label: "Revoke Invitation",
          value: "revert_invitation",
        },
      ]);
    });

    it("returns the correct options for an enabled user", () => {
      expect(userActionOptions(false, userStub, false)).toEqual([
        { disabled: false, label: "Disable Account", value: "disable_account" },
        { disabled: false, label: "Promote User", value: "promote_user" },
        {
          disabled: false,
          label: "Require Password Reset",
          value: "reset_password",
          helpText:
            "This will revoke all active Fleet API tokens for this user.",
        },
        { disabled: false, label: "Modify Details", value: "modify_details" },
      ]);
    });

    it("returns the correct options for a disabled user", () => {
      const disabledUser = { ...userStub, enabled: false };

      expect(userActionOptions(false, disabledUser, false)).toEqual([
        { disabled: false, label: "Enable Account", value: "enable_account" },
        { disabled: false, label: "Promote User", value: "promote_user" },
        {
          disabled: false,
          label: "Require Password Reset",
          value: "reset_password",
          helpText:
            "This will revoke all active Fleet API tokens for this user.",
        },
        { disabled: false, label: "Modify Details", value: "modify_details" },
      ]);
    });

    it("returns the correct options for an admin", () => {
      const adminUser = { ...userStub, admin: true };

      expect(userActionOptions(false, adminUser, false)).toEqual([
        { disabled: false, label: "Disable Account", value: "disable_account" },
        { disabled: false, label: "Demote User", value: "demote_user" },
        {
          disabled: false,
          label: "Require Password Reset",
          value: "reset_password",
          helpText:
            "This will revoke all active Fleet API tokens for this user.",
        },
        { disabled: false, label: "Modify Details", value: "modify_details" },
      ]);
    });

    it("returns the correct options for the current user", () => {
      const adminUser = { ...userStub, admin: true };

      expect(userActionOptions(true, adminUser, false)).toEqual([
        { disabled: true, label: "Disable Account", value: "disable_account" },
        { disabled: true, label: "Demote User", value: "demote_user" },
        {
          disabled: false,
          label: "Require Password Reset",
          value: "reset_password",
          helpText:
            "This will revoke all active Fleet API tokens for this user.",
        },
        { disabled: false, label: "Modify Details", value: "modify_details" },
      ]);
    });
  });

  describe("#userStatusLabel", () => {
    const { userStatusLabel } = helpers;

    it("returns the correct options for an invite", () => {
      expect(userStatusLabel(userStub, true)).toEqual("Invited");
    });

    it("returns the correct options for an enabled user", () => {
      expect(userStatusLabel(userStub, false)).toEqual("Active");
    });

    it("returns the correct options for a disabled user", () => {
      const disabledUser = { ...userStub, enabled: false };

      expect(userStatusLabel(disabledUser, false)).toEqual("Disabled");
    });
  });
});
