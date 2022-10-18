import React, { useState, useEffect, useContext } from "react";
import { InjectedRouter } from "react-router";
import { size } from "lodash";

import paths from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { RoutingContext } from "context/routing";
import { ISSOSettings } from "interfaces/ssoSettings";
import local from "utilities/local";
import sessionsAPI from "services/entities/sessions";
import formatErrorResponse from "utilities/format_error_response";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
// @ts-ignore
import LoginForm from "components/forms/LoginForm";
import { AxiosError } from "axios";

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
    currentUser,
    setAvailableTeams,
    setCurrentUser,
    setCurrentTeam,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const { redirectLocation } = useContext(RoutingContext);
  const [loginVisible, setLoginVisible] = useState(true);
  const [ssoSettings, setSSOSettings] = useState<ISSOSettings>();
  const [pageStatus, setPageStatus] = useState<string | null>(
    new URLSearchParams(location.search).get("status")
  );
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  useEffect(() => {
    const { HOME } = paths;
    const getSSO = async () => {
      try {
        const { settings } = await sessionsAPI.ssoSettings();
        setSSOSettings(settings);
      } catch (error) {
        console.error(error);
        return false;
      }
    };

    if (!currentUser) {
      getSSO();
    }

    if (currentUser && !currentUser.force_password_reset) {
      router?.push(HOME);
    }

    if (pageStatus && pageStatus in statusMessages) {
      renderFlash("error", statusMessages[pageStatus as keyof IStatusMessages]);
    }
  }, [router, currentUser]);

  const onChange = () => {
    if (size(errors)) {
      setErrors({});
    }

    return false;
  };

  const onSubmit = async (formData: ILoginData) => {
    const { HOME, RESET_PASSWORD } = paths;

    try {
      const { user, available_teams, token } = await sessionsAPI.create(
        formData
      );
      local.setItem("auth_token", token);

      setLoginVisible(false);
      setCurrentUser(user);
      setAvailableTeams(available_teams);
      setCurrentTeam(undefined);

      // Redirect to password reset page if user is forced to reset password.
      // Any other requests will fail.
      if (user.force_password_reset) {
        return router.push(RESET_PASSWORD);
      }
      return router.push(redirectLocation || HOME);
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      setErrors(errorObject);
      return false;
    }
  };

  const ssoSignOn = async () => {
    const { HOME } = paths;
    let returnToAfterAuth = HOME;
    if (redirectLocation != null) {
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
  };

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
