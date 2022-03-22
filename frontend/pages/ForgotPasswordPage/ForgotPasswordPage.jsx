import React, { Component } from "react";

import PATHS from "router/paths";
import usersAPI from "services/entities/users";

import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import debounce from "utilities/debounce";
import ForgotPasswordForm from "components/forms/ForgotPasswordForm";
import StackedWhiteBoxes from "components/StackedWhiteBoxes";

export class ForgotPasswordPage extends Component {
  constructor() {
    super();

    this.state = {
      email: null,
      errors: {},
    };
  }

  componentWillUnmount() {
    return this.clearErrors();
  }

  handleSubmit = debounce(async (formData) => {
    try {
      await usersAPI.forgotPassword(formData);

      const { email } = formData;
      this.setState({ email, errors: {} });
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      this.setState({ email: null, errors: errorObject });
      return false;
    }
  });

  clearErrors = () => {
    this.setState({ errors: {} });
  };

  renderContent = () => {
    const { clearErrors, handleSubmit } = this;
    const { email, errors } = this.state;
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
        onChangeFunc={clearErrors}
        serverErrors={errors}
      />
    );
  };

  render() {
    const leadText =
      "Enter your email below and we will email you a link so that you can reset your password.";

    return (
      <StackedWhiteBoxes
        leadText={leadText}
        previousLocation={PATHS.LOGIN}
        className="forgot-password"
      >
        {this.renderContent()}
      </StackedWhiteBoxes>
    );
  }
}

export default ForgotPasswordPage;
