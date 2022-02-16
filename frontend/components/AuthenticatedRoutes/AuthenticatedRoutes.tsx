import React from "react";
import { push } from "react-router-redux";
import { useDispatch, useSelector } from "react-redux";

import paths from "router/paths";
import { IRedirectLocation } from "interfaces/redirect_location"; // @ts-ignore
import { setRedirectLocation } from "redux/nodes/redirectLocation/actions";
import { IUser } from "interfaces/user";
import { useDeepEffect } from "utilities/hooks";
import { authToken } from "utilities/local";

interface IAppProps {
  children: JSX.Element;
  location: any; // no type in react-router v3
}

interface IRootState {
  auth: {
    user: IUser;
  };
  routing: {
    locationBeforeTransitions: IRedirectLocation;
  };
}

export const AuthenticatedRoutes = ({ children, location }: IAppProps) => {
  const dispatch = useDispatch();
  const { user } = useSelector((state: IRootState) => state.auth);
  const { locationBeforeTransitions } = useSelector(
    (state: IRootState) => state.routing
  );

  const redirectToLogin = () => {
    const { LOGIN } = paths;

    dispatch(setRedirectLocation(locationBeforeTransitions));
    return dispatch(push(LOGIN));
  };

  const redirectToPasswordReset = () => {
    const { RESET_PASSWORD } = paths;

    return dispatch(push(RESET_PASSWORD));
  };

  const redirectToApiUserOnly = () => {
    const { API_ONLY_USER } = paths;

    return dispatch(push(API_ONLY_USER));
  };

  useDeepEffect(() => {
    // this works with App.tsx. if authToken does
    // exist, user state is checked and fetched if null
    if (!authToken()) {
      return redirectToLogin();
    }

    if (user && user.force_password_reset) {
      return redirectToPasswordReset();
    }

    if (user && user.api_only) {
      return redirectToApiUserOnly();
    }
  }, [user]);

  useDeepEffect(() => {
    window.scrollTo(0, 0);
  }, [location]);

  if (!user) {
    return false;
  }

  return <div>{children}</div>;
};

export default AuthenticatedRoutes;
