import React, { useState, useEffect, useContext, useCallback } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { size } from "lodash";
import { AxiosError } from "axios";

import paths from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { RoutingContext } from "context/routing";
import { ISSOSettings } from "interfaces/ssoSettings";
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

interface ILoginData {
  email: string;
  password: string;
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

  const pageStatus = new URLSearchParams(location.search).get("status");
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    pageStatus && pageStatus in statusMessages
      ? formatErrorResponse({
          status: pageStatus,
          data: { errors: [{ name: "base", reason: "Authentication failed" }] },
        })
      : {}
  );
  const [loginVisible, setLoginVisible] = useState(true);

  const {
    data: ssoSettings,
    isLoading: isLoadingSSOSettings,
    error: errorSSOSettings,
  } = useQuery<ISSOSettingsResponse, Error, ISSOSettings>(
    ["ssoSettings"],
    () => sessionsAPI.ssoSettings(),
    {
      enabled: !currentUser,
      onError: (err) => {
        console.error(err);
      },
      select: (data) => data.settings,
    }
  );

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

  useEffect(() => {
    // this only needs to run once so we can wrap it in useEffect to avoid unneccesary third-party
    // API calls
    (async function testGravatarAvailability() {
      try {
        const response = await fetch("https://gravatar.com/avatar");
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

  // // TODO(sarah): fix this effect so that it isn't causing infinte re-renders
  // useEffect(() => {
  //   if (pageStatus && pageStatus in statusMessages) {
  //     renderFlash("error", statusMessages[pageStatus as keyof IStatusMessages]);
  //   }
  // }, [pageStatus, renderFlash]);

  const onChange = useCallback(() => {
    if (size(errors)) {
      setErrors({});
    }

    return false;
  }, [errors]);

  const onSubmit = useCallback(
    async (formData: ILoginData) => {
      const { DASHBOARD, RESET_PASSWORD } = paths;

      try {
        const { user, available_teams, token } = await sessionsAPI.create(
          formData
        );
        local.setItem("auth_token", token);

        setLoginVisible(false);
        setCurrentUser(user);
        setAvailableTeams(user, available_teams);
        setCurrentTeam(undefined);

        // Redirect to password reset page if user is forced to reset password.
        // Any other requests will fail.
        if (user.force_password_reset) {
          return router.push(RESET_PASSWORD);
        }

        if (!config) {
          const configResponse = await configAPI.loadAll();
          setConfig(configResponse);
        }
        return router.push(redirectLocation || DASHBOARD);
      } catch (response) {
        const errorObject = formatErrorResponse(response);
        setErrors(errorObject);
        return false;
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
      console.log("url", url);
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

  console.log("errors", errors);

  if (isLoadingSSOSettings) {
    return <Spinner />;
  }

  return (
    <AuthenticationFormWrapper>
      <LoginForm
        onChangeFunc={onChange}
        handleSubmit={onSubmit}
        isHidden={!loginVisible}
        serverErrors={errors}
        ssoSettings={ssoSettings}
        handleSSOSignOn={ssoSignOn}
      />
    </AuthenticationFormWrapper>
  );
};

export default LoginPage;
