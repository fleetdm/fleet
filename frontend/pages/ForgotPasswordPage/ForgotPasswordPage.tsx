import React, { useEffect, useState } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";
import usersAPI from "services/entities/users";
import formatErrorResponse from "utilities/format_error_response";

// @ts-ignore
import ForgotPasswordForm from "components/forms/ForgotPasswordForm";
// @ts-ignore
import StackedWhiteBoxes from "components/StackedWhiteBoxes";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";

interface IForgotPasswordPage {
  router: InjectedRouter;
}

const ForgotPasswordPage = ({ router }: IForgotPasswordPage) => {
  const [email, setEmail] = useState("");
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [isLoading, setIsLoading] = useState(false);

  const baseClass = "forgot-password";

  useEffect(() => {
    setErrors({});
  }, []);

  const handleSubmit = async (formData: any) => {
    setIsLoading(true);
    try {
      await usersAPI.forgotPassword(formData);

      setEmail(formData.email);
      setErrors({});
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      setEmail("");
      setErrors(errorObject);
      return false;
    } finally {
      setIsLoading(false);
    }
  };

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    } else if (email) {
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
              url="https://fleetdm.com/docs/using-fleet/fleetctl-cli#using-fleetctl-with-an-api-only-user?utm_medium=fleetui&utm_campaign=get-api-token"
              text="Password reset FAQ"
              newTab
            />
          </p>
        </div>
      );
    }

    return (
      <>
        <p>
          Enter your email below to receive an email with instructions to reset
          your password.
        </p>
        <ForgotPasswordForm
          handleSubmit={handleSubmit}
          onChangeFunc={() => setErrors({})}
          serverErrors={errors}
        />
      </>
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

export default ForgotPasswordPage;
