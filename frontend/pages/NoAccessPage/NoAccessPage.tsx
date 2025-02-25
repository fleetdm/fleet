// Page returned when a user has no access because they have no global or team role

import React, { useEffect } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

import { CONTACT_FLEET_LINK } from "utilities/constants";

import Button from "components/buttons/Button/Button";
// @ts-ignore
import StackedWhiteBoxes from "components/StackedWhiteBoxes";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import CustomLink from "components/CustomLink/CustomLink";

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
    <AuthenticationFormWrapper>
      <StackedWhiteBoxes
        router={router}
        headerText="This account does not currently have access to Fleet."
      >
        <>
          <p>
            To get access,{" "}
            <CustomLink
              url={orgContactUrl || CONTACT_FLEET_LINK}
              text="contact your administrator"
            />
            .
          </p>
          <Button
            variant="brand"
            onClick={onBackToLogin}
            className={`${baseClass}__btn`}
          >
            Back to login
          </Button>
        </>
      </StackedWhiteBoxes>
    </AuthenticationFormWrapper>
  );
};

export default NoAccessPage;
