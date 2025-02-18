import React, { useState, useEffect, useContext, useCallback } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";

import paths from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { RoutingContext } from "context/routing";
import { ISSOSettings } from "interfaces/ssoSettings";
import { ILoginUserData } from "interfaces/user";
import local from "utilities/local";
import configAPI from "services/entities/config";
import sessionsAPI, { ISSOSettingsResponse } from "services/entities/sessions";
import formatErrorResponse from "utilities/format_error_response";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
// @ts-ignore
import LoginForm from "components/forms/LoginForm";
import Spinner from "components/Spinner/Spinner";

interface ILoginPageProps {
  router: InjectedRouter; // v3
  location: {
    pathname: string;
    query: { vulnerable?: boolean };
    search: string;
  };
}

interface IStatusMessages {
  account_disabled: string;
  account_invalid: string;
  org_disabled: string;
  error: string;
}

const statusMessages: IStatusMessages = {
  account_disabled:
    "Single sign-on is not enabled on your account. Please contact your Fleet administrator.",
  account_invalid: "You do not have a Fleet account.",
  org_disabled: "Single sign-on is not enabled for your organization.",
  error:
    "There was an error with single sign-on. Please contact your Fleet administrator.",
};

const baseClass = "login-page";

const LoginPage = ({ router, location }: ILoginPageProps) => {
  const {
    availableTeams,
    config,
    currentUser,
    setAvailableTeams,
    setConfig,
    setCurrentUser,
    setCurrentTeam,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const { redirectLocation } = useContext(RoutingContext);

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [pendingEmail, setPendingEmail] = useState(false);

  const { data: ssoSettings, isLoading: isLoadingSSOSettings } = useQuery<
    ISSOSettingsResponse,
    Error,
    ISSOSettings
  >(["ssoSettings"], () => sessionsAPI.ssoSettings(), {
    enabled: !currentUser,
    onError: (err) => {
      console.error(err);
    },
    select: (data) => data.settings,
  });

  useEffect(() => {
    // this only needs to run once so we can wrap it in useEffect to avoid unneccesary third-party
    // API calls
    (async function testGravatarAvailability() {
      try {
        const response = await fetch("https://www.gravatar.com/avatar/0000");
        if (response.ok) {
          localStorage.setItem("gravatar_available", "true");
        } else {
          localStorage.setItem("gravatar_available", "false");
        }
      } catch (error) {
        localStorage.setItem("gravatar_available", "false");
      }
    })();
  }, []);

  useEffect(() => {
    if (
      availableTeams &&
      config &&
      currentUser &&
      !currentUser.force_password_reset
    ) {
      router.push(redirectLocation || paths.DASHBOARD);
    }
  }, [availableTeams, config, currentUser, redirectLocation, router]);

  // TODO: Fix this. If renderFlash is added as a dependency it causes infinite re-renders.
  useEffect(() => {
    let status = new URLSearchParams(location.search).get("status");
    status = status && statusMessages[status as keyof IStatusMessages];
    if (status) {
      renderFlash("error", status);
    }
  }, [location?.search]);

  const onSubmit = useCallback(
    async (formData: ILoginUserData) => {
      setIsSubmitting(true);
      const { DASHBOARD, RESET_PASSWORD, NO_ACCESS } = paths;

      try {
        const response = await sessionsAPI.login(formData);
        const { user, available_teams, token } = response;

        local.setItem("auth_token", token);

        setCurrentUser(user);
        setAvailableTeams(user, available_teams);
        setCurrentTeam(undefined);

        if (!user.global_role && user.teams.length === 0) {
          return router.push(NO_ACCESS);
        }
        // Redirect to password reset page if user is forced to reset password.
        // Any other requests will fail.
        else if (user.force_password_reset) {
          return router.push(RESET_PASSWORD);
        } else if (!config) {
          const configResponse = await configAPI.loadAll();
          setConfig(configResponse);
        }
        return router.push(redirectLocation || DASHBOARD);
      } catch (response) {
        if ((response as { status: number }).status === 202) {
          setPendingEmail(true);
        }

        const errorObject = formatErrorResponse(response);
        setErrors(errorObject);
        return false;
      } finally {
        setIsSubmitting(false);
      }
    },
    [
      config,
      redirectLocation,
      router,
      setAvailableTeams,
      setConfig,
      setCurrentTeam,
      setCurrentUser,
    ]
  );

  const ssoSignOn = useCallback(async () => {
    const { DASHBOARD } = paths;
    let returnToAfterAuth = DASHBOARD;
    if (redirectLocation !== null) {
      returnToAfterAuth = redirectLocation;
    }

    try {
      const { url } = await sessionsAPI.initializeSSO(returnToAfterAuth);
      window.location.href = url;
    } catch (error) {
      const err = error as AxiosError;
      // a one-off error for sso login failure to be more readable to users
      const ssoError = {
        status: err.status,
        data: { errors: [{ name: "base", reason: "Authentication failed" }] },
      };
      const errorObject = formatErrorResponse(ssoError);
      setErrors(errorObject);
      return false;
    }
  }, [redirectLocation]);

  if (isLoadingSSOSettings) {
    return <Spinner className={`${baseClass}__loading-spinner`} />;
  }

  return (
    <AuthenticationFormWrapper>
      <LoginForm
        handleSubmit={onSubmit}
        baseError={errors.base}
        ssoSettings={ssoSettings}
        handleSSOSignOn={ssoSignOn}
        isSubmitting={isSubmitting}
        pendingEmail={pendingEmail}
      />
    </AuthenticationFormWrapper>
  );
};

export default LoginPage;
