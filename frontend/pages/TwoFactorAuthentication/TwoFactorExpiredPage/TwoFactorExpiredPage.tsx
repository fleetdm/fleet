import React, { useState } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

// @ts-ignore
import StackedWhiteBoxes from "components/StackedWhiteBoxes";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import Spinner from "components/Spinner";
import BackLink from "components/BackLink";
import Button from "components/buttons/Button";

interface ITwoFactorExpiredPage {
  router: InjectedRouter;
}

const TwoFactorExpiredPage = ({ router }: ITwoFactorExpiredPage) => {
  // TODO: pushing here after clicking an expired link
  const [isLoading, setIsLoading] = useState(false);

  const baseClass = "two-factor-expired";

  const onClickLoginButton = () => {
    router.push(PATHS.LOGIN);
  };

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }
    return (
      <>
        <p>
          {/* NEED TO COMPLETE EMAIL ADDRESS */}
          <b>That link is expired.</b>
        </p>
        <p>Log in again for a new link.</p>
        <Button variant="brand" onClick={onClickLoginButton}>
          Back to login
        </Button>
      </>
    );
  };

  return (
    <AuthenticationFormWrapper>
      <StackedWhiteBoxes router={router} className={baseClass}>
        {renderContent()}
      </StackedWhiteBoxes>
    </AuthenticationFormWrapper>
  );
};

export default TwoFactorExpiredPage;
