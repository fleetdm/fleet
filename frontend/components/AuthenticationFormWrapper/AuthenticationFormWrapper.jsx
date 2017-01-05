import React, { Component, PropTypes } from 'react';

const baseClass = 'auth-form-wrapper';

class AuthenticationFormWrapper extends Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render () {
    const { children } = this.props;

    return (
      <div className={baseClass}>
        <img alt="Kolide text logo" src="/assets/images/kolide-logo-text.svg" className={`${baseClass}__logo`} />
        {children}
      </div>
    );
  }
}

export default AuthenticationFormWrapper;
