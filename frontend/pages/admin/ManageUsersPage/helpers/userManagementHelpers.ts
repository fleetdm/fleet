import { isEqual } from "lodash";

import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import { IApiError } from "interfaces/errors";
import { IInvite } from "interfaces/invite";
import { IUser, IUpdateUserFormData } from "interfaces/user";
import { IFormErrors } from "hooks/useFormValidation";
import { IUserFormData } from "../components/UserForm/UserForm";

type ICurrentUserData = Pick<
  IUser,
  "global_role" | "teams" | "name" | "email" | "sso_enabled" | "mfa_enabled"
>;

export interface IRoleOptionsParams {
  isPremiumTier?: boolean;
  isApiOnly?: boolean;
}

/**
 * Helper function that will compare the current user with data from the editing
 * form and return an object with the difference between the two. This can be
 * be used for PATCH updates when updating a user.
 * @param currentUserData
 * @param formData
 */
const generateUpdateData = (
  currentUserData: IUser | IInvite,
  formData: IUserFormData
): IUpdateUserFormData => {
  const updatableFields = [
    "global_role",
    "teams",
    "name",
    "email",
    "sso_enabled",
    "mfa_enabled",
  ];
  return Object.keys(formData).reduce<IUpdateUserFormData>(
    (updatedAttributes, attr) => {
      const key = attr as keyof ICurrentUserData;
      // attribute can be updated and is different from the current value.
      if (
        updatableFields.includes(attr) &&
        !isEqual(formData[key], currentUserData[key])
      ) {
        (updatedAttributes as Record<string, unknown>)[attr] = formData[key];
      }
      return updatedAttributes;
    },
    {}
  );
};

export const roleOptions = ({
  isPremiumTier,
  isApiOnly,
}: IRoleOptionsParams): CustomOptionType[] => {
  const roles: CustomOptionType[] = [
    {
      label: "Observer",
      value: "observer",
    },
    {
      label: "Maintainer",
      value: "maintainer",
    },
    {
      label: "Admin",
      value: "admin",
    },
  ];

  if (isPremiumTier) {
    roles.splice(1, 0, {
      label: "Observer+",
      value: "observer_plus",
    });

    roles.splice(2, 0, {
      label: "Technician",
      value: "technician",
    });

    if (isApiOnly) {
      roles.splice(3, 0, {
        label: "GitOps",
        value: "gitops",
      });
    }
  }

  return roles;
};

/**
 * Maps a users/invites API failure to inline field errors, or null when the
 * failure isn't field-specific and belongs in a toast instead.
 */
export const getUserFieldErrors = (userErrors: {
  data: IApiError;
}): IFormErrors | null => {
  const reason = userErrors.data.errors?.[0]?.reason ?? "";

  // Check the invite-specific wording first: it also contains "already exists".
  if (
    reason.includes("already invited") ||
    (reason.includes("Invite") && reason.includes("already exists"))
  ) {
    return { email: "Enter an email that hasn't already been invited" };
  }
  if (reason.includes("already exists") || reason.includes("Duplicate")) {
    return { email: "Enter an email that isn't already in use" };
  }
  if (reason.includes("required criteria")) {
    return { password: "Enter a password that meets the requirements below" };
  }
  if (reason.includes("password too long")) {
    return { password: "Enter a password with 48 characters or fewer" };
  }
  return null;
};

export default {
  generateUpdateData,
  roleOptions,
  getUserFieldErrors,
};
