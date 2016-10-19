import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';

import componentStyles from './styles';
import {
  clearForgotPasswordErrors,
  forgotPasswordAction,
} from '../../redux/nodes/components/ForgotPasswordPage/actions';
import debounce from '../../utilities/debounce';
import ForgotPasswordForm from '../../components/forms/ForgotPasswordForm';
import Icon from '../../components/icons/Icon';
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

  onSubmit = debounce((formData) => {
    const { dispatch } = this.props;

    return dispatch(forgotPasswordAction(formData));
  })

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
    const leadText = 'If you’ve forgotten your password enter your email below and we will email you a link so that you can reset your password.';
    const whiteBoxOverrideStyles = {
      headerWrapper: { textAlign: 'left' },
    };

    return (
      <StackedWhiteBoxes
        headerText="Forgot Password"
        leadText={leadText}
        previousLocation="/login"
        style={whiteBoxOverrideStyles}
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
