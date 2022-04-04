import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import paths from "router/paths";
import { AppContext } from "context/app";
import { RoutingContext } from "context/routing";
import { useDeepEffect } from "utilities/hooks";
import { authToken } from "utilities/local";

interface IAppProps {
  children: JSX.Element;
  location: any; // no type in react-router v3
  router: InjectedRouter;
}

export const AuthenticatedRoutes = ({ children, location, router }: IAppProps) => {
  const { setRedirectLocation } = useContext(RoutingContext);
  const { currentUser } = useContext(AppContext);

  const redirectToLogin = () => {
    const { LOGIN } = paths;

    setRedirectLocation(window.location.pathname);
    return router.push(LOGIN);
  };

  const redirectToPasswordReset = () => {
    const { RESET_PASSWORD } = paths;

    return router.push(RESET_PASSWORD);
  };

  const redirectToApiUserOnly = () => {
    const { API_ONLY_USER } = paths;

    return router.push(API_ONLY_USER);
  };

  useDeepEffect(() => {
    // this works with App.tsx. if authToken does
    // exist, user state is checked and fetched if null
    if (!authToken()) {
      return redirectToLogin();
    }

    if (currentUser?.force_password_reset && !authToken()) {
      return redirectToPasswordReset();
    }

    if (currentUser?.api_only) {
      return redirectToApiUserOnly();
    }
  }, [currentUser]);

  useDeepEffect(() => {
    window.scrollTo(0, 0);
  }, [location]);

  if (!currentUser) {
    return false;
  }

  return <div>{children}</div>;
};

export default AuthenticatedRoutes;
