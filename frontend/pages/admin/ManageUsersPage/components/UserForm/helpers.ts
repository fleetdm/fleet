import { ITeam } from "interfaces/team";
import { UserRole } from "interfaces/user";
import { IFormErrors } from "hooks/useFormValidation";
import validatePresence from "components/forms/validators/validate_presence";
import validEmail from "components/forms/validators/valid_email";
import validPassword from "components/forms/validators/valid_password";

export enum NewUserType {
  AdminInvited = "ADMIN_INVITED",
  AdminCreated = "ADMIN_CREATED",
}

export type UserFormState = {
  email: string;
  name: string;
  newUserType: NewUserType | null;
  password: string;
  sso_enabled: boolean;
  mfa_enabled: boolean;
  global_role: UserRole | null;
  teams: ITeam[];
};

export interface IUserFormValidationContext {
  isNewUser: boolean;
  isInvitePending?: boolean;
  isSsoEnabled?: boolean;
  isPremiumTier: boolean;
  isEmailReadOnly: boolean;
}

const PASSWORD_ERRORS: Record<string, string> = {
  too_short: "Enter a password with at least 12 characters",
  too_long: "Enter a password with 48 characters or fewer",
  invalid_format: "Enter a password with at least 1 number and 1 symbol",
};

export const isPasswordShown = (
  data: UserFormState,
  { isNewUser, isInvitePending }: IUserFormValidationContext
): boolean =>
  ((isNewUser && data.newUserType !== NewUserType.AdminInvited) ||
    (!isNewUser && !isInvitePending)) &&
  !data.sso_enabled;

// A password is required when creating a user with SSO off, and when moving a
// user off SSO onto password auth (which includes the case where SSO was turned
// off org-wide). On an edit form for a password user it stays optional: leaving
// it blank keeps the current password.
const isPasswordRequired = (
  data: UserFormState,
  { isNewUser, isSsoEnabled }: IUserFormValidationContext
): boolean =>
  (isNewUser && data.newUserType === NewUserType.AdminCreated) ||
  !!isSsoEnabled;

export const validateUserForm = (
  data: UserFormState,
  context: IUserFormValidationContext
): IFormErrors => {
  const errors: IFormErrors = {};

  if (!validatePresence(data.name)) {
    errors.name = "Enter a name";
  }

  if (!context.isEmailReadOnly) {
    if (!validatePresence(data.email)) {
      errors.email = "Enter an email";
    } else if (!validEmail(data.email)) {
      errors.email = "Enter a valid email";
    }
  }

  if (isPasswordShown(data, context)) {
    const hasPassword = validatePresence(data.password);
    if (!hasPassword && isPasswordRequired(data, context)) {
      errors.password = "Enter a password";
    } else if (hasPassword) {
      // An optional password still has to satisfy the format rule once it has a
      // value. Fall back to the validator's own copy if it grows a new code.
      const { error_code: errorCode, error } = validPassword(data.password);
      if (errorCode) {
        errors.password = PASSWORD_ERRORS[errorCode] || error;
      }
    }
  }

  if (context.isPremiumTier && !data.global_role && !data.teams.length) {
    errors.teams = "Select at least one fleet";
  }

  return errors;
};
