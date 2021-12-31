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
const AuthAnyAdminRoutes = ({
  children,
}: IAuthAnyAdminRoutesProps): JSX.Element | null => {
  const dispatch = useDispatch();
  const user = useSelector((state: IRootState) => state.auth.user);

  if (!user) {
    return null;
  }

  const teamId = Number(children.props.params.team_id) || null;
  let allowAccess;

  if (teamId && user.teams) {
    const userAdminTeams = user.teams.filter(
      (thisTeam) => thisTeam.role === "admin"
    );
    allowAccess = userAdminTeams.some((thisTeam) => thisTeam.id === teamId);
  }

  if (
    (!permissionUtils.isGlobalAdmin(user) &&
      !permissionUtils.isAnyTeamAdmin(user)) ||
    (!permissionUtils.isGlobalAdmin(user) && !allowAccess)
  ) {
    dispatch(push(HOME));
    dispatch(renderFlash("error", "You do not have permissions for that page"));
    return null;
  }
  return <>{children}</>;
};

export default AuthAnyAdminRoutes;
