import React, { useState, useEffect, useContext } from "react";
import { InjectedRouter } from "react-router";

import paths from "router/paths";
import { AppContext } from "context/app"; // @ts-ignore
import sessionsAPI from "services/entities/sessions";
import local from "utilities/local"; // @ts-ignore
import { formatErrorResponse } from "redux/nodes/entities/base/helpers"

// @ts-ignore
import LoginSuccessfulPage from "pages/LoginSuccessfulPage"; // @ts-ignore
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper"; // @ts-ignore
import LoginForm from "components/forms/LoginForm"; // @ts-ignore

interface ILoginPreviewPageProps {
  router: InjectedRouter; // v3
}
interface ILoginData {
  email: string;
  password: string;
}

const PreviewLoginPage = ({ router }: ILoginPreviewPageProps): JSX.Element => {
  const { isPreviewMode, setAvailableTeams, setCurrentUser, setCurrentTeam } = useContext(
    AppContext
  );
  const [loginVisible, setLoginVisible] = useState<boolean>(true);

  const onSubmit = async (formData: ILoginData) => {
    const { HOME } = paths;

    try {
      const { user, available_teams, token } = await sessionsAPI.create(formData)
      local.setItem("auth_token", token);

      setLoginVisible(false);
      setCurrentUser(user);
      setAvailableTeams(available_teams);
      setCurrentTeam(undefined);

      return router.push(HOME);
    } catch (response) {
      console.error(response);
      return false;
    }
  };

  useEffect(() => {
    if (isPreviewMode) {
      onSubmit({
        email: "admin@example.com",
        password: "admin123#",
      });
    }
  }, []);

  return (
    <AuthenticationFormWrapper>
      <LoginSuccessfulPage />
      <LoginForm handleSubmit={onSubmit} isHidden={!loginVisible} />
    </AuthenticationFormWrapper>
  );
};

export default PreviewLoginPage;
