import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { push } from "react-router-redux";
import { noop } from "lodash";

import {
  clearForgotPasswordErrors,
  forgotPasswordAction,
} from "redux/nodes/components/ForgotPasswordPage/actions";
import debounce from "utilities/debounce";
import ForgotPasswordForm from "components/forms/ForgotPasswordForm";
import StackedWhiteBoxes from "components/StackedWhiteBoxes";

export class ForgotPasswordPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    email: PropTypes.string,
    errors: PropTypes.shape({
      base: PropTypes.string,
    }),
  };

  static defaultProps = {
    dispatch: noop,
  };

  componentWillUnmount() {
    return this.clearErrors();
  }

  handleLeave = (location) => {
    const { dispatch } = this.props;

    return dispatch(push(location));
  };

  handleSubmit = debounce((formData) => {
    const { dispatch } = this.props;

    return dispatch(forgotPasswordAction(formData)).catch(() => false);
  });

  clearErrors = () => {
    const { dispatch } = this.props;

    return dispatch(clearForgotPasswordErrors);
  };

  renderContent = () => {
    const { clearErrors, handleSubmit } = this;
    const { email, errors } = this.props;

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
    const { handleLeave } = this;
    const leadText =
      "Enter your email below and we will email you a link so that you can reset your password.";

    return (
      <StackedWhiteBoxes
        leadText={leadText}
        previousLocation="/login"
        className="forgot-password"
        onLeave={handleLeave}
      >
        {this.renderContent()}
      </StackedWhiteBoxes>
    );
  }
}

const mapStateToProps = (state) => {
  return state.components.ForgotPasswordPage;
};

export default connect(mapStateToProps)(ForgotPasswordPage);
