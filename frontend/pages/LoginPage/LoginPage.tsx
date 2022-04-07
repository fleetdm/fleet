import React, { useState, useEffect, useContext } from "react";
import { InjectedRouter } from "react-router";
import { size } from "lodash";

import paths from "router/paths";
import { AppContext } from "context/app";
import { RoutingContext } from "context/routing";
import { ISSOSettings } from "interfaces/ssoSettings";
import local from "utilities/local";
import sessionsAPI from "services/entities/sessions";
import formatErrorResponse from "utilities/format_error_response";

// @ts-ignore
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper"; // @ts-ignore
import LoginForm from "components/forms/LoginForm"; // @ts-ignore
import LoginSuccessfulPage from "pages/LoginSuccessfulPage"; // @ts-ignore

interface ILoginPageProps {
  router: InjectedRouter; // v3
}

interface ILoginData {
  email: string;
  password: string;
}

const LoginPage = ({ router }: ILoginPageProps) => {
  const {
    currentUser,
    setAvailableTeams,
    setCurrentUser,
    setCurrentTeam,
  } = useContext(AppContext);
  const { redirectLocation } = useContext(RoutingContext);
  const [loginVisible, setLoginVisible] = useState<boolean>(true);
  const [ssoSettings, setSSOSettings] = useState<ISSOSettings>();
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

    if (currentUser) {
      router?.push(HOME);
    } else {
      getSSO();
    }
  }, [router]);

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
      console.error(error);
      return false;
    }
  };

  return (
    <AuthenticationFormWrapper>
      <LoginSuccessfulPage />
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
