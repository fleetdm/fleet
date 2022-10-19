import React, { useEffect, useState } from "react";

import PATHS from "router/paths";
import usersAPI from "services/entities/users";
import formatErrorResponse from "utilities/format_error_response";

// @ts-ignore
import ForgotPasswordForm from "components/forms/ForgotPasswordForm";
// @ts-ignore
import StackedWhiteBoxes from "components/StackedWhiteBoxes";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import Spinner from "components/Spinner";
import ExternalLinkIcon from "../../../assets/images/icon-external-link-12x12@2x.png";

const ForgotPasswordPage = () => {
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
            <a
              href="https://fleetdm.com/docs/using-fleet/fleetctl-cli#using-fleetctl-with-an-api-only-user?utm_medium=fleetui&utm_campaign=get-api-token"
              target="_blank"
              rel="noopener noreferrer"
              className="external-link"
            >
              Password reset FAQ
              <img src={ExternalLinkIcon} alt="Open external link" />
            </a>
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
      <StackedWhiteBoxes previousLocation={PATHS.LOGIN}>
        <div className={baseClass}>{renderContent()}</div>
      </StackedWhiteBoxes>
    </AuthenticationFormWrapper>
  );
};

export default ForgotPasswordPage;
