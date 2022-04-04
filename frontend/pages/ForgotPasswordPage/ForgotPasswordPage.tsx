import React, { useEffect, useState } from "react";

import PATHS from "router/paths";
import usersAPI from "services/entities/users";

// @ts-ignore
import { formatErrorResponse } from "redux/nodes/entities/base/helpers"; // @ts-ignore
import ForgotPasswordForm from "components/forms/ForgotPasswordForm"; // @ts-ignore
import StackedWhiteBoxes from "components/StackedWhiteBoxes"; // @ts-ignore
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";

const ForgotPasswordPage = () => {
  const [email, setEmail] = useState<string>("");
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  useEffect(() => {
    setErrors({});
  }, []);

  const handleSubmit = async (formData: any) => {
    try {
      await usersAPI.forgotPassword(formData);

      setEmail(formData.email);
      setErrors({});
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      setEmail("");
      setErrors(errorObject);
      return false;
    }
  };

  const renderContent = () => {
    const baseClass = "forgot-password";

    if (email) {
      return (
        <div>
          <div className={`${baseClass}__text-wrapper`}>
            <p className={`${baseClass}__text`}>
              An email was sent to
              <span className={`${baseClass}__email`}> {email}</span>. Click the
              link on the email to proceed with the password reset process.
            </p>
          </div>
        </div>
      );
    }

    return (
      <ForgotPasswordForm
        handleSubmit={handleSubmit}
        onChangeFunc={() => setErrors({})}
        serverErrors={errors}
      />
    );
  };

  const leadText =
    "Enter your email below and we will email you a link so that you can reset your password.";

  return (
    <AuthenticationFormWrapper>
      <StackedWhiteBoxes
        leadText={leadText}
        previousLocation={PATHS.LOGIN}
        className="forgot-password"
      >
        {renderContent()}
      </StackedWhiteBoxes>
    </AuthenticationFormWrapper>
  );
};

export default ForgotPasswordPage;
