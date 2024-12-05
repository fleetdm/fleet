import React, { useEffect, useContext } from "react";
import { InjectedRouter } from "react-router";

import paths from "router/paths";
import { AppContext } from "context/app";
import sessionsAPI from "services/entities/sessions";
import local from "utilities/local";

import LoginSuccessfulPage from "pages/LoginSuccessfulPage";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
// @ts-ignore
import LoginForm from "components/forms/LoginForm";

interface ILoginPreviewPageProps {
  router: InjectedRouter; // v3
}
interface ILoginData {
  email: string;
  password: string;
}

const LoginPreviewPage = ({ router }: ILoginPreviewPageProps): JSX.Element => {
  const {
    isPreviewMode,
    setAvailableTeams,
    setCurrentUser,
    setCurrentTeam,
  } = useContext(AppContext);

  const onSubmit = async (formData: ILoginData) => {
    const { DASHBOARD } = paths;

    try {
      const { user, available_teams, token } = await sessionsAPI.login(
        formData
      );
      local.setItem("auth_token", token);

      setCurrentUser(user);
      setAvailableTeams(user, available_teams);
      setCurrentTeam(undefined);

      return router.push(DASHBOARD);
    } catch (response) {
      console.error(response);
      return false;
    }
  };

  useEffect(() => {
    if (isPreviewMode) {
      onSubmit({
        email: "admin@example.com",
        password: "preview1337#",
      });
    }
  }, []);

  return (
    <AuthenticationFormWrapper>
      <LoginSuccessfulPage />
      <LoginForm
        handleSubmit={onSubmit}
        isSubmitting={false} // TODO fix
        pendingEmail={false} // TODO fix
      />
    </AuthenticationFormWrapper>
  );
};

export default LoginPreviewPage;
