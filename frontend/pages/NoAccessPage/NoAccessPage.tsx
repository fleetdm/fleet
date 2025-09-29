// Page returned when a user has no access because they have no global or team role

import React, { useEffect } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

import { CONTACT_FLEET_LINK } from "utilities/constants";

import Button from "components/buttons/Button/Button";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import CustomLink from "components/CustomLink/CustomLink";
import AuthenticationNav from "components/AuthenticationNav";

const baseClass = "no-access-page";

interface INoAccessPageProps {
  router: InjectedRouter;
  orgContactUrl?: string;
}

const NoAccessPage = ({ router, orgContactUrl }: INoAccessPageProps) => {
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
    <AuthenticationFormWrapper
      header="Access denied"
      headerCta={
        <AuthenticationNav router={router} previousLocation={PATHS.LOGIN} />
      }
      className={baseClass}
    >
      <div className={`${baseClass}__description`}>
        <p>
          This account does not currently have access to Fleet.
          <br />
          To get access,{" "}
          <CustomLink
            url={orgContactUrl || CONTACT_FLEET_LINK}
            text="contact your administrator"
          />
          .
        </p>
      </div>
      <Button onClick={onBackToLogin} className={`${baseClass}__btn`}>
        Back to login
      </Button>
    </AuthenticationFormWrapper>
  );
};

export default NoAccessPage;
