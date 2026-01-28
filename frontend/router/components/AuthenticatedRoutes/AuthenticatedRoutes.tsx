import React, { useContext, useEffect, useRef } from "react";
import { InjectedRouter } from "react-router";

import paths from "router/paths";
import { AppContext } from "context/app";
import { RoutingContext } from "context/routing";
import useDeepEffect from "hooks/useDeepEffect";
import { authToken, clearToken } from "utilities/local";
import { useErrorHandler } from "react-error-boundary";
import permissions from "utilities/permissions";

interface IAppProps {
  children: JSX.Element;
  location: any; // no type in react-router v3
  router: InjectedRouter;
}

export const AuthenticatedRoutes = ({
  children,
  location,
  router,
}: IAppProps) => {
  // used to ensure single pendo intialization
  const isPendoIntialized = useRef(false);

  const { setRedirectLocation } = useContext(RoutingContext);
  const { currentUser, config, isSandboxMode } = useContext(AppContext);

  const handlePageError = useErrorHandler();

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

  // used for pendo intialisation.
  // run only on sandbox mode when pendo has not been initialized
  // and we have values for the current user and config
  useEffect(() => {
    if (isSandboxMode && !isPendoIntialized.current && currentUser && config) {
      const { email, name } = currentUser;
      const {
        org_info: { org_name },
        server_settings: { server_url },
      } = config;

      // @ts-ignore
      window?.pendo?.initialize({
        visitor: {
          id: email,
          email,
          full_name: name,
        },
        account: {
          id: server_url,
          name: org_name,
        },
      });

      isPendoIntialized.current = true;
    }
  }, [isSandboxMode, currentUser, config]);

  useDeepEffect(() => {
    // this works with App.tsx. if authToken does
    // exist, user state is checked and fetched if null
    if (!authToken()) {
      if (window.location.hostname.includes(".sandbox.fleetdm.com")) {
        window.location.href = "https://www.fleetdm.com/try-fleet/login";
      }

      return redirectToLogin();
    }

    if (currentUser?.force_password_reset && !authToken()) {
      return redirectToPasswordReset();
    }

    if (currentUser?.api_only) {
      return redirectToApiUserOnly();
    }

    if (currentUser && permissions.isNoAccess(currentUser)) {
      clearToken();
      return handlePageError({ status: 403 });
    }
  }, [currentUser]);

  useDeepEffect(() => {
    if (location.hash) {
      const elementToScrollTo = location.hash.slice(1);

      setTimeout(() => {
        document
          .getElementById(elementToScrollTo)
          ?.scrollIntoView({ behavior: "smooth", block: "start" });
      }, 100);
    }

    window.scrollTo(0, 0);
  }, [location]);

  if (!currentUser) {
    return false;
  }

  return <div>{children}</div>;
};

export default AuthenticatedRoutes;
