import { isEqual } from "lodash";

import { IInvite } from "interfaces/invite";
import { IUser, IUserUpdateBody } from "interfaces/user";
import { IFormData } from "../components/UserForm/UserForm";

type ICurrentUserData = Pick<
  IUser,
  "global_role" | "teams" | "name" | "email" | "sso_enabled"
>;

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
): IUserUpdateBody => {
  const updatableFields = [
    "global_role",
    "teams",
    "name",
    "email",
    "sso_enabled",
  ];
  return Object.keys(formData).reduce<IUserUpdateBody>(
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

export default {
  generateUpdateData,
};
