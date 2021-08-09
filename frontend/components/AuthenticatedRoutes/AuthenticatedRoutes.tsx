import React, { useState } from "react";
import { push } from "react-router-redux";
import { useDispatch, useSelector } from "react-redux";

import paths from "router/paths";
import { IRedirectLocation } from "interfaces/redirect_location"; // @ts-ignore
import { setRedirectLocation } from "redux/nodes/redirectLocation/actions";
import { IUser } from "interfaces/user";
import { useDeepEffect } from "utilities/hooks";

interface IAppProps { 
  children: JSX.Element;
};

interface IRootState {
  auth: {
    user: IUser;
    loading: boolean;
  };
  routing: {
    locationBeforeTransitions: IRedirectLocation;
  };
}

export const AuthenticatedRoutes = ({ children }: IAppProps) => {
  const dispatch = useDispatch();
  const { loading, user } = useSelector((state: IRootState) => state.auth); 
  const { locationBeforeTransitions } = useSelector((state: IRootState) => state.routing); 

  const [mulligan, setMulligan] = useState(false);

  useDeepEffect(() => {
    // TODO: refreshing the page always begins with `loading` and `user` false.
    // In a nutshell, Redux is not playing nice with a functional component in App.tsx
    // because the fetchCurrentUser() call doesn't make it in time, so `mulligan` helps for now
    if (!loading && !user && !mulligan) {
      setMulligan(true);
      return; 
    }

    if (!loading && !user && mulligan) {
      return redirectToLogin();
    }

    if (user && user.force_password_reset) {
      return redirectToPasswordReset();
    }

    if (user && user.api_only) {
      return redirectToApiUserOnly();
    }

    return () => {};
  }, [loading, user]);

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

  if (!user) {
    return false;
  }

  return <div>{children}</div>;
}

export default AuthenticatedRoutes;
