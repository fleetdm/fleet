import React, { useEffect, useState, useContext } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import usersAPI from "services/entities/users";
import configAPI from "services/entities/config";
import formatErrorResponse from "utilities/format_error_response";

// @ts-ignore
import ResetPasswordForm from "components/forms/ResetPasswordForm";
// @ts-ignore
import StackedWhiteBoxes from "components/StackedWhiteBoxes";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import NoAccessPage from "pages/NoAccessPage/NoAccessPage";

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

  const continueWithLoggedInUser = async (formData: any) => {
    const { new_password } = formData;

    try {
      await usersAPI.performRequiredPasswordReset(new_password as string);
      const config = await configAPI.loadAll();
      setConfig(config);
      return router.push(PATHS.DASHBOARD);
    } catch (response: any) {
      const errorObject = formatErrorResponse(response);
      setErrors(errorObject);
      console.log("errorObject", errorObject);
      if (
        response.data.message.includes(
          "either global role or team role needs to be defined"
        )
      ) {
        console.log("hitting the right error");
        setErrors({ no_access: "No global role or team role" });
      }
      return false;
    }
  };

  const onSubmit = async (formData: any) => {
    if (currentUser) {
      return continueWithLoggedInUser(formData);
    }

    console.log("currentUser", currentUser);

    const resetPasswordData = {
      ...formData,
      password_reset_token: token,
    };

    console.log("resetPasswordData", resetPasswordData);

    try {
      await usersAPI.resetPassword(resetPasswordData);
      router.push(PATHS.LOGIN);
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      setErrors(errorObject);
      return false;
    }
  };

  if (errors.no_access === "No global role or team role") {
    return <NoAccessPage router={router} />;
  }

  return (
    <AuthenticationFormWrapper>
      <StackedWhiteBoxes leadText="Create a new password. Your new password must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)">
        <ResetPasswordForm handleSubmit={onSubmit} serverErrors={errors} />
      </StackedWhiteBoxes>
    </AuthenticationFormWrapper>
  );
};

export default ResetPasswordPage;
