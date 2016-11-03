import React, { Component, PropTypes } from 'react';

class AuthenticationFormWrapper extends Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render () {
    const { children } = this.props;

    return (
      <div className="auth-form-wrapper">
        <img alt="Kolide text logo" src="/assets/images/kolide-logo-text.svg" />
        {children}
      </div>
    );
  }
}

export default AuthenticationFormWrapper;
