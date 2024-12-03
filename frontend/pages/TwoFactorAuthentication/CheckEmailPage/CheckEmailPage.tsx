import React, { useState } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
// @ts-ignore
import StackedWhiteBoxes from "components/StackedWhiteBoxes";
import Spinner from "components/Spinner";
import BackLink from "components/BackLink";

interface ICheckEmailPage {
  router: InjectedRouter;
}

const CheckEmailPage = ({ router }: ICheckEmailPage) => {
  const [isLoading, setIsLoading] = useState(false);
  // TODO: pushing here instead of to the app

  const baseClass = "two-factor-email-check";

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }
    return (
      <div className={`${baseClass}__text-wrapper`}>
        <BackLink text="Back to login" path={PATHS.LOGIN} />
        <h1>Check your email</h1>
        <p className={`${baseClass}__text`}>
          {/* NEED TO COMPLETE EMAIL ADDRESS */}
          We sent an email to you at <b>TODO</b>. <br />
          Please click the magic link in the email to sign in.
        </p>
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

export default CheckEmailPage;
