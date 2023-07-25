import React from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

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
  return (
    <AuthenticationFormWrapper>
      <StackedWhiteBoxes headerText="This account does not currently have access to Fleet.">
        <>
          <p>
            To get access,{" "}
            <CustomLink
              url={orgContactUrl || "https://fleetdm.com/contact"}
              text="contact your administrator"
            />
            .
          </p>
          <Button
            variant="brand"
            onClick={() => {
              router.push(PATHS.LOGIN);
            }}
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
