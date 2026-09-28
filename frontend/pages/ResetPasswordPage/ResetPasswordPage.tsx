import React, { useEffect, useContext } from "react";
import { InjectedRouter } from "react-router";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import ResetPasswordForm from "components/forms/ResetPasswordForm";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import PATHS from "router/paths";
import configAPI from "services/entities/config";
import usersAPI from "services/entities/users";
import formatErrorResponse from "utilities/format_error_response";

const baseClass = "reset-password-page";
interface IResetPasswordPageProps {
  location: {
    query: { token?: string };
  };
  router: InjectedRouter;
}

const ResetPasswordPage = ({ location, router }: IResetPasswordPageProps) => {
  const { token } = location.query;
  const { currentUser, setConfig } = useContext(AppContext);

  useEffect(() => {
    if (!currentUser && !token) {
      router.push(PATHS.LOGIN);
    }
  }, [currentUser, token]);

  // No access prompt if currentUser data has no role
  useEffect(() => {
    if (!currentUser?.global_role && currentUser?.teams.length === 0) {
      router.push(PATHS.NO_ACCESS);
    }
  }, [currentUser]);

  const continueWithLoggedInUser = async (formData: any) => {
    const { new_password } = formData;

    try {
      await usersAPI.performRequiredPasswordReset(new_password as string);
      const config = await configAPI.loadAll();
      setConfig(config);
      return router.push(PATHS.DASHBOARD);
    } catch (response: any) {
      if (
        response.data.message.includes(
          "either global role or team role needs to be defined"
        )
      ) {
        router.push(PATHS.NO_ACCESS);
        return false;
      }
      const { base } = formatErrorResponse(response);
      notify.error(base || "Couldn't reset your password. Try again.", {
        response,
      });
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
      const { base } = formatErrorResponse(response);
      notify.error(base || "Couldn't reset your password. Try again.", {
        response,
      });
      return false;
    }
  };

  return (
    <AuthenticationFormWrapper className={baseClass}>
      <div className={`${baseClass}__description`}>
        <p>
          Create a new password. Your new password must include 12-48
          characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol
          (e.g. &*#)
        </p>
      </div>
      <ResetPasswordForm handleSubmit={onSubmit} />
    </AuthenticationFormWrapper>
  );
};

export default ResetPasswordPage;
