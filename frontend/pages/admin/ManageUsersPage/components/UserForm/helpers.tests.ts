import createMockTeam from "__mocks__/teamMock";

import {
  IUserFormValidationContext,
  NewUserType,
  UserFormState,
  validateUserForm,
} from "./helpers";

const VALID: UserFormState = {
  email: "user@example.com",
  name: "User 1",
  newUserType: NewUserType.AdminCreated,
  password: "pa55word!1234",
  sso_enabled: false,
  mfa_enabled: false,
  global_role: "observer",
  teams: [],
};

const NEW_USER_CONTEXT: IUserFormValidationContext = {
  isNewUser: true,
  isPremiumTier: true,
  isEmailReadOnly: false,
};

const EDIT_USER_CONTEXT: IUserFormValidationContext = {
  isNewUser: false,
  isPremiumTier: true,
  isEmailReadOnly: false,
};

describe("validateUserForm", () => {
  it("returns no errors for a valid new user", () => {
    expect(validateUserForm(VALID, NEW_USER_CONTEXT)).toEqual({});
  });

  it("requires a name", () => {
    expect(
      validateUserForm({ ...VALID, name: "   " }, NEW_USER_CONTEXT).name
    ).toBe("Enter a name");
  });

  it("prefers the presence error over the format error on email", () => {
    expect(
      validateUserForm({ ...VALID, email: "" }, NEW_USER_CONTEXT).email
    ).toBe("Enter an email");
    expect(
      validateUserForm({ ...VALID, email: "nope" }, NEW_USER_CONTEXT).email
    ).toBe("Enter a valid email");
  });

  it("skips the email rules when the field is read-only", () => {
    const errors = validateUserForm(
      { ...VALID, email: "nope" },
      {
        ...EDIT_USER_CONTEXT,
        isEmailReadOnly: true,
      }
    );

    expect(errors.email).toBeUndefined();
  });

  describe("password requirement", () => {
    it("requires one for a new admin-created user", () => {
      expect(
        validateUserForm({ ...VALID, password: "" }, NEW_USER_CONTEXT).password
      ).toBe("Enter a password");
    });

    it("does not require one when inviting a new user", () => {
      const errors = validateUserForm(
        { ...VALID, password: "", newUserType: NewUserType.AdminInvited },
        NEW_USER_CONTEXT
      );

      expect(errors.password).toBeUndefined();
    });

    it("does not require one when SSO is on", () => {
      const errors = validateUserForm(
        { ...VALID, password: "", sso_enabled: true },
        NEW_USER_CONTEXT
      );

      expect(errors.password).toBeUndefined();
    });

    it("leaves it optional on an edit form for a password user", () => {
      const errors = validateUserForm(
        { ...VALID, password: "" },
        EDIT_USER_CONTEXT
      );

      expect(errors.password).toBeUndefined();
    });

    it("requires one when moving an existing SSO user onto a password", () => {
      const errors = validateUserForm(
        { ...VALID, password: "" },
        {
          ...EDIT_USER_CONTEXT,
          isSsoEnabled: true,
        }
      );

      expect(errors.password).toBe("Enter a password");
    });

    it("does not require one on a pending invite being edited", () => {
      const errors = validateUserForm(
        { ...VALID, password: "" },
        {
          ...EDIT_USER_CONTEXT,
          isInvitePending: true,
        }
      );

      expect(errors.password).toBeUndefined();
    });
  });

  describe("password format", () => {
    it.each([
      ["short1!", "Enter a password with at least 12 characters"],
      [`${"a1!".repeat(16)}x`, "Enter a password with 48 characters or fewer"],
      ["abcdefghijkl", "Enter a password with at least 1 number and 1 symbol"],
    ])("rejects %s", (password, expected) => {
      expect(
        validateUserForm({ ...VALID, password }, NEW_USER_CONTEXT).password
      ).toBe(expected);
    });

    it("checks the format of an optional password that has a value", () => {
      const errors = validateUserForm(
        { ...VALID, password: "short1!" },
        EDIT_USER_CONTEXT
      );

      expect(errors.password).toBe(
        "Enter a password with at least 12 characters"
      );
    });
  });

  describe("fleet assignment", () => {
    it("requires at least one fleet for a non-global user", () => {
      expect(
        validateUserForm(
          { ...VALID, global_role: null, teams: [] },
          NEW_USER_CONTEXT
        ).teams
      ).toBe("Select at least one fleet");
    });

    it("accepts a non-global user with a fleet", () => {
      const errors = validateUserForm(
        { ...VALID, global_role: null, teams: [createMockTeam({ id: 1 })] },
        NEW_USER_CONTEXT
      );

      expect(errors.teams).toBeUndefined();
    });

    it("does not apply on free tier, where the selector isn't rendered", () => {
      const errors = validateUserForm(
        { ...VALID, global_role: null, teams: [] },
        { ...NEW_USER_CONTEXT, isPremiumTier: false }
      );

      expect(errors.teams).toBeUndefined();
    });
  });
});
