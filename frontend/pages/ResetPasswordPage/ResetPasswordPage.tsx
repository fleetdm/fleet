import React, { useEffect, useState, useContext } from "react";
import { InjectedRouter } from "react-router";
import { size } from "lodash";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import usersAPI from "services/entities/users";
import configAPI from "services/entities/config";
import formatErrorResponse from "utilities/format_error_response";

// @ts-ignore
import ResetPasswordForm from "components/forms/ResetPasswordForm"; // @ts-ignore
import StackedWhiteBoxes from "components/StackedWhiteBoxes"; // @ts-ignore
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";

interface IResetPasswordPageProps {
  location: any; // no type in react-router v3
  router: InjectedRouter;
}

const ResetPasswordPage = ({ location, router }: IResetPasswordPageProps) => {
  const { token } = location.query;
  const { currentUser, setConfig } = useContext(AppContext);
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  useEffect(() => {
    if (!currentUser && !token) {
      router.push(PATHS.LOGIN);
    }
  }, [currentUser, token]);

  const onResetErrors = () => {
    if (size(errors)) {
      setErrors({});
    }
  };

  const continueWithLoggedInUser = async (formData: any) => {
    const { new_password } = formData;

    try {
      await usersAPI.performRequiredPasswordReset(new_password as string);
      const config = await configAPI.loadAll();
      setConfig(config);
      return router.push(PATHS.HOME);
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      setErrors(errorObject);
      return false;
    }
  };

  const onSubmit = async (formData: any) => {
    if (currentUser) {
      return continueWithLoggedInUser(formData);
    }

    const resetPasswordData = {
      ...formData,
      password_reset_token: token,
    };

    try {
      await usersAPI.resetPassword(resetPasswordData);
      router.push(PATHS.LOGIN);
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      setErrors(errorObject);
      return false;
    }
  };

  return (
    <AuthenticationFormWrapper>
      <StackedWhiteBoxes leadText="Create a new password. Your new password must include 7 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)">
        <ResetPasswordForm
          handleSubmit={onSubmit}
          onChangeFunc={onResetErrors}
          serverErrors={errors}
        />
      </StackedWhiteBoxes>
    </AuthenticationFormWrapper>
  );
};

export default ResetPasswordPage;
