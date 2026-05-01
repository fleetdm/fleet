// TODO: Clean up/remove isPreviewMode

import React, { useCallback, useEffect, useContext } from "react";
import { InjectedRouter } from "react-router";

import paths from "router/paths";
import { AppContext } from "context/app";
import sessionsAPI from "services/entities/sessions";
import authToken from "utilities/auth_token";

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

  const onSubmit = useCallback(
    async (formData: ILoginData) => {
      const { DASHBOARD } = paths;

      try {
        const {
          user,
          available_teams,
          token,
          token_expires_at,
        } = await sessionsAPI.login(formData);
        const expiresAt = token_expires_at
          ? new Date(token_expires_at)
          : undefined;
        authToken.save(token, expiresAt);

        setCurrentUser(user);
        setAvailableTeams(user, available_teams);
        setCurrentTeam(undefined);

        return router.push(DASHBOARD);
      } catch (response) {
        console.error(response);
        return false;
      }
    },
    [router, setAvailableTeams, setCurrentTeam, setCurrentUser]
  );

  useEffect(() => {
    if (isPreviewMode) {
      onSubmit({
        email: "admin@example.com",
        password: "preview1337#",
      });
    }
  }, [isPreviewMode, onSubmit]);

  return (
    <AuthenticationFormWrapper header="Login successful">
      <p>Taking you to the Fleet UI...</p>
      <LoginForm
        handleSubmit={onSubmit}
        isSubmitting={false}
        pendingEmail={false}
      />
    </AuthenticationFormWrapper>
  );
};

export default LoginPreviewPage;
