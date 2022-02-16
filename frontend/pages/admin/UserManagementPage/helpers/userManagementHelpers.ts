import {
  isEmpty,
  isEqual,
  isPlainObject,
  isString,
  reduce,
  trim,
  union,
} from "lodash";

import { IInvite } from "interfaces/invite";
import { IUser, IUserUpdateBody } from "interfaces/user";
import { IFormData } from "../components/UserForm/UserForm";

type ICurrentUserData = Pick<
  IUser,
  "global_role" | "teams" | "name" | "email" | "sso_enabled"
>;

interface ILocationParams {
  pathPrefix?: string;
  routeTemplate?: string;
  routeParams?: { [key: string]: any };
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

export const getNextLocationPath = ({
  pathPrefix = "",
  routeTemplate = "",
  routeParams = {},
}: ILocationParams): string => {
  const pathPrefixFinal = isString(pathPrefix) ? pathPrefix : "";
  const routeTemplateFinal = (isString(routeTemplate) && routeTemplate) || "";
  const routeParamsFinal = isPlainObject(routeParams) ? routeParams : {};

  let routeString = "";

  if (!isEmpty(routeParamsFinal)) {
    routeString = reduce(
      routeParamsFinal,
      (string, value, key) => {
        return string.replace(`:${key}`, encodeURIComponent(value));
      },
      routeTemplateFinal
    );
  }

  const nextLocation = union(
    trim(pathPrefixFinal, "/").split("/"),
    routeString.split("/")
  ).join("/");

  return `/${nextLocation}`;
};

export default {
  generateUpdateData,
  getNextLocationPath,
};
