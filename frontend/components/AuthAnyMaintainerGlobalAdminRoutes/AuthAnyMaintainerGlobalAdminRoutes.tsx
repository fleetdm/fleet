import React from "react";
import { useDispatch, useSelector } from "react-redux";
import { push } from "react-router-redux";

import { IUser } from "interfaces/user";
import permissionUtils from "utilities/permissions";
import paths from "router/paths";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

interface IAuthAnyMaintainerGlobalAdminRoutesProps {
  children: JSX.Element;
}

interface IRootState {
  auth: {
    user: IUser;
  };
}

const { HOME } = paths;

/**
 * Checks if a user is a global admin or global maintainer when routing
 */
const AuthAnyMaintainerGlobalAdminRoutes = (
  props: IAuthAnyMaintainerGlobalAdminRoutesProps
): JSX.Element | null => {
  const { children } = props;

  const dispatch = useDispatch();
  const user = useSelector((state: IRootState) => state.auth.user);

  if (!user) {
    return null;
  }

  if (
    !permissionUtils.isGlobalAdmin(user) &&
    !permissionUtils.isGlobalMaintainer(user) &&
    !permissionUtils.isAnyTeamMaintainer(user)
  ) {
    dispatch(push(HOME));
    dispatch(renderFlash("error", "You do not have permissions for that page"));
    return null;
  }
  return <>{children}</>;
};

export default AuthAnyMaintainerGlobalAdminRoutes;
