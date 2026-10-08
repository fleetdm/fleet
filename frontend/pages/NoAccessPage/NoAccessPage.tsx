// Page returned when a user has no access because they have no global or team role

import React, { useEffect } from "react";
import { InjectedRouter } from "react-router";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import Button from "components/buttons/Button/Button";
import PATHS from "router/paths";

const baseClass = "no-access-page";

interface INoAccessPageProps {
  router: InjectedRouter;
}

const NoAccessPage = ({ router }: INoAccessPageProps) => {
  const onBackToLogin = () => {
    router.push(PATHS.LOGIN);
  };

  useEffect(() => {
    if (onBackToLogin) {
      const closeOrSaveWithEnterKey = (event: KeyboardEvent) => {
        if (event.code === "Enter" || event.code === "NumpadEnter") {
          event.preventDefault();
          onBackToLogin();
        }
      };

      document.addEventListener("keydown", closeOrSaveWithEnterKey);
      return () => {
        document.removeEventListener("keydown", closeOrSaveWithEnterKey);
      };
    }
  }, [onBackToLogin]);

  return (
    <AuthenticationFormWrapper header="Access denied" className={baseClass}>
      <div className={`${baseClass}__description`}>
        <p>
          This account does not currently have access to Fleet.
          <br />
          To get access, contact your administrator.
        </p>
      </div>
      <div className="button-wrap--center">
        <Button onClick={onBackToLogin} size="wide">
          Back to login
        </Button>
      </div>
    </AuthenticationFormWrapper>
  );
};

export default NoAccessPage;
