import React, { useState } from "react";
import { InjectedRouter } from "react-router";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import AuthenticationNav from "components/AuthenticationNav";
import CustomLink from "components/CustomLink";
import ForgotPasswordForm, {
  IForgotPasswordFormData,
} from "components/forms/ForgotPasswordForm/ForgotPasswordForm";
import { notify } from "components/ToastNotification";
import PATHS from "router/paths";
import usersAPI from "services/entities/users";
import formatErrorResponse from "utilities/format_error_response";

interface IForgotPasswordPage {
  router: InjectedRouter;
}

const ForgotPasswordPage = ({ router }: IForgotPasswordPage) => {
  const [email, setEmail] = useState("");

  const baseClass = "forgot-password";

  const handleSubmit = async (formData: IForgotPasswordFormData) => {
    try {
      await usersAPI.forgotPassword(formData);
      setEmail(formData.email);
    } catch (response) {
      setEmail("");
      const { base } = formatErrorResponse(response);
      notify.error(base || "Couldn't send the reset email. Try again.", {
        response,
      });
    }
  };

  const renderContent = () => {
    if (email) {
      return (
        <div className={`${baseClass}__text-wrapper`}>
          <p className={`${baseClass}__text`}>
            An email was sent to{" "}
            <span className={`${baseClass}__email`}>{email}</span>. Click the
            link in the email to proceed with the password reset process. If you
            did not receive an email please contact your Fleet administrator.
            <br />
            <br />
            You can find more information on resetting passwords at the{" "}
            <CustomLink
              url="https://fleetdm.com/docs/using-fleet/fleetctl-cli?utm_medium=fleetui&utm_campaign=get-api-token#using-fleetctl-with-an-api-only-user"
              text="Password reset FAQ"
              newTab
            />
          </p>
        </div>
      );
    }

    return <ForgotPasswordForm handleSubmit={handleSubmit} />;
  };

  return (
    <AuthenticationFormWrapper
      header="Reset password"
      headerCta={
        <AuthenticationNav previousLocation={PATHS.LOGIN} router={router} />
      }
      className={baseClass}
    >
      {renderContent()}
    </AuthenticationFormWrapper>
  );
};

export default ForgotPasswordPage;
