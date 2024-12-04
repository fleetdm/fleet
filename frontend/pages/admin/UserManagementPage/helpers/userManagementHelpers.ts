import { isEqual } from "lodash";

import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import { IInvite } from "interfaces/invite";
import { IUser, IUserUpdateBody, IUpdateUserFormData } from "interfaces/user";
import { IFormData } from "../components/UserForm/UserForm";

type ICurrentUserData = Pick<
  IUser,
  "global_role" | "teams" | "name" | "email" | "sso_enabled"
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
  formData: IFormData
): IUpdateUserFormData => {
  const updatableFields = [
    "global_role",
    "teams",
    "name",
    "email",
    "sso_enabled",
  ];
  return Object.keys(formData).reduce<IUserUpdateBody | any>(
    (updatedAttributes, attr) => {
      // attribute can be updated and is different from the current value.
      if (
        updatableFields.includes(attr) &&
        !isEqual(
          formData[attr as keyof ICurrentUserData],
          currentUserData[attr as keyof ICurrentUserData]
        )
      ) {
        // Note: ignore TS error as we will never have undefined set to an
        // updatedAttributes value if we get to this code.
        // @ts-ignore
        updatedAttributes[attr as keyof ICurrentUserData] =
          formData[attr as keyof ICurrentUserData];
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

    if (isApiOnly) {
      roles.splice(3, 0, {
        label: "GitOps",
        value: "gitops",
      });
    }
  }

  return roles;
};

export default {
  generateUpdateData,
  roleOptions,
};
