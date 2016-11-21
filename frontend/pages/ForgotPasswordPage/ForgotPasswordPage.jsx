import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';

import {
  clearForgotPasswordErrors,
  forgotPasswordAction,
} from '../../redux/nodes/components/ForgotPasswordPage/actions';
import debounce from '../../utilities/debounce';
import ForgotPasswordForm from '../../components/forms/ForgotPasswordForm';
import StackedWhiteBoxes from '../../components/StackedWhiteBoxes';

export class ForgotPasswordPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    email: PropTypes.string,
    error: PropTypes.string,
  };

  static defaultProps = {
    dispatch: noop,
  };

  handleSubmit = debounce((formData) => {
    const { dispatch } = this.props;

    return dispatch(forgotPasswordAction(formData));
  })

  clearErrors = () => {
    const { dispatch } = this.props;

    return dispatch(clearForgotPasswordErrors);
  }

  renderContent = () => {
    const { clearErrors, handleSubmit } = this;
    const { email, error } = this.props;

    const baseClass = 'forgot-password';

    if (email) {
      return (
        <div>
          <div className={`${baseClass}__text-wrapper`}>
            <p className={`${baseClass}__text`}>
              An email was sent to
              <span className={`${baseClass}__email`}> {email}</span>.
               Click the link on the email to proceed with the password reset process.
            </p>
          </div>
          <div className={`${baseClass}__button`}>
            <i className={`${baseClass}__icon kolidecon kolidecon-success-check`} />
            EMAIL SENT
          </div>
        </div>
      );
    }

    return (
      <ForgotPasswordForm
        onChangeFunc={clearErrors}
        errors={{ email: error }}
        handleSubmit={handleSubmit}
      />
    );
  }

  render () {
    const leadText = 'If you’ve forgotten your password enter your email below and we will email you a link so that you can reset your password.';

    return (
      <StackedWhiteBoxes
        headerText="Forgot Password"
        leadText={leadText}
        previousLocation="/login"
        className="forgot-password__header"
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
