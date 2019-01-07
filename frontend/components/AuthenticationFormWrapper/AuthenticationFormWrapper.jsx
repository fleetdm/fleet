import React, { Component } from 'react';
import PropTypes from 'prop-types';

const baseClass = 'auth-form-wrapper';

class AuthenticationFormWrapper extends Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render () {
    const { children } = this.props;

    return (
      <div className={baseClass}>
        <img alt="Kolide Fleet" src="/assets/images/kolide-logo-vertical.svg" className={`${baseClass}__logo`} />
        {children}
      </div>
    );
  }
}

export default AuthenticationFormWrapper;
