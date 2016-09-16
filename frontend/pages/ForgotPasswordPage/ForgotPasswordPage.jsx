import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import componentStyles from './styles';
import {
  clearForgotPasswordErrors,
  forgotPasswordAction,
} from '../../redux/nodes/components/ForgotPasswordPage/actions';
import ForgotPasswordForm from '../../components/forms/ForgotPasswordForm';
import Icon from '../../components/icons/Icon';

export class ForgotPasswordPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    email: PropTypes.string,
    error: PropTypes.string,
  };

  static defaultProps = {
    dispatch: noop,
  };

  onSubmit = (formData) => {
    const { dispatch } = this.props;

    return dispatch(forgotPasswordAction(formData));
  }

  clearErrors = () => {
    const { dispatch } = this.props;

    return dispatch(clearForgotPasswordErrors);
  }

  renderContent = () => {
    const { clearErrors } = this;
    const { email, error } = this.props;
    const {
      emailSentButtonWrapperStyles,
      emailSentIconStyles,
      emailSentTextStyles,
      emailSentTextWrapperStyles,
      emailTextStyles,
    } = componentStyles;

    if (email) {
      return (
        <div>
          <div style={emailSentTextWrapperStyles}>
            <p style={emailSentTextStyles}>
              An email was sent to
              <span style={emailTextStyles}> {email}</span>.
               Click the link on the email to proceed with the password reset process.
            </p>
          </div>
          <div style={emailSentButtonWrapperStyles}>
            <Icon name="check" style={emailSentIconStyles} />
            EMAIL SENT
          </div>
        </div>
      );
    }

    return (
      <ForgotPasswordForm
        clearErrors={clearErrors}
        error={error}
        onSubmit={this.onSubmit}
      />
    );
  }

  render () {
    const {
      containerStyles,
      forgotPasswordStyles,
      headerStyles,
      smallWhiteTabStyles,
      textStyles,
      whiteTabStyles,
    } = componentStyles;

    return (
      <div style={containerStyles}>
        <div style={smallWhiteTabStyles} />
        <div style={whiteTabStyles} />
        <div style={forgotPasswordStyles}>
          <p style={headerStyles}>Forgot Password</p>
          <p style={textStyles}>If you’ve forgotten your password enter your email below and we will email you a link so that you can reset your password.</p>
          {this.renderContent()}
        </div>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  return state.components.ForgotPasswordPage;
};

export default connect(mapStateToProps)(ForgotPasswordPage);
