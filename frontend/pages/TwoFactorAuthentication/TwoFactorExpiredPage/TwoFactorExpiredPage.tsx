import React, { useState } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

// @ts-ignore
import StackedWhiteBoxes from "components/StackedWhiteBoxes";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import Spinner from "components/Spinner";
import BackLink from "components/BackLink";
import Button from "components/buttons/Button";

interface ITwoFactorExpiredLink {
  router: InjectedRouter;
}

const TwoFactorExpiredLink = ({ router }: ITwoFactorExpiredLink) => {
  // TODO: pushing here after clicking an expired link
  const [isLoading, setIsLoading] = useState(false);

  const baseClass = "two-factor-expired-link";

  const onClickLoginButton = () => {
    router.push(PATHS.LOGIN);
  };

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }
    return (
      <div className={`${baseClass}__text-wrapper`}>
        <BackLink text="Back to login" path={PATHS.LOGIN} />
        <p className={`${baseClass}__text`}>
          {/* NEED TO COMPLETE EMAIL ADDRESS */}
          <b>That link is expired.</b> <br />
          Log in again for a new link.
        </p>
        <Button variant="brand" onClick={onClickLoginButton}>
          Back to login
        </Button>
      </div>
    );
  };

  return (
    <AuthenticationFormWrapper>
      <StackedWhiteBoxes previousLocation={PATHS.LOGIN} router={router}>
        <div className={baseClass}>{renderContent()}</div>
      </StackedWhiteBoxes>
    </AuthenticationFormWrapper>
  );
};

export default TwoFactorExpiredLink;
