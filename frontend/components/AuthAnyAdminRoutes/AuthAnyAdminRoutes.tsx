import React from "react";
import { useDispatch, useSelector } from "react-redux";
import { push } from "react-router-redux";

import { IUser } from "interfaces/user";
import permissionUtils from "utilities/permissions";
import paths from "router/paths";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

interface IAuthAnyAdminRoutesProps {
  children: JSX.Element;
}

interface IRootState {
  auth: {
    user: IUser;
  };
}

const { HOME } = paths;

/**
 * Checks if a user is any admin when routing
 */
const AuthAnyAdminRoutes = (
  props: IAuthAnyAdminRoutesProps
): JSX.Element | null => {
  const { children } = props;

  const dispatch = useDispatch();
  const user = useSelector((state: IRootState) => state.auth.user);

  if (!user) {
    return null;
  }

  if (
    !permissionUtils.isGlobalAdmin(user) &&
    !permissionUtils.isAnyTeamAdmin(user)
  ) {
    dispatch(push(HOME));
    dispatch(renderFlash("error", "You do not have permissions for that page"));
    return null;
  }
  return <>{children}</>;
};

export default AuthAnyAdminRoutes;
