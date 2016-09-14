import React, { Component } from 'react';
import componentStyles from './styles';
import ForgotPasswordForm from '../../components/forms/ForgotPasswordForm';

class ForgotPasswordPage extends Component {
  onSubmit = (formData) => {
    console.log('ForgotPasswordPage formData', formData);
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
          <ForgotPasswordForm onSubmit={this.onSubmit} />
        </div>
      </div>
    );
  }
}

export default ForgotPasswordPage;
