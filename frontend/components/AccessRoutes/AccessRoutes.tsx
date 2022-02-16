import React from "react";
import { useDispatch, useSelector } from "react-redux";
import { push } from "react-router-redux";

import { IUser } from "interfaces/user";
import permissionUtils from "utilities/permissions";
import paths from "router/paths";

interface IAccessRoutes {
  children: JSX.Element;
}

interface IRootState {
  auth: {
    user: IUser;
  };
}

const { FLEET_403 } = paths;

const AccessRoutes = ({ children }: IAccessRoutes): JSX.Element | null => {
  const dispatch = useDispatch();
  const user = useSelector((state: IRootState) => state.auth.user);

  // user is an empty object here. The API result has not come back
  // so render nothing.
  if (Object.keys(user).length === 0) {
    return null;
  }

  if (permissionUtils.isNoAccess(user)) {
    dispatch(push(FLEET_403));
    return null;
  }
  return <>{children}</>;
};

export default AccessRoutes;
