import React, { Component } from 'react';
import PropTypes from 'prop-types';

import logoVertical from '../../../assets/images/kolide-logo-vertical.svg';

const baseClass = 'auth-form-wrapper';

class AuthenticationFormWrapper extends Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render () {
    const { children } = this.props;

    return (
      <div className={baseClass}>
        <img alt="Kolide Fleet" src={logoVertical} className={`${baseClass}__logo`} />
        {children}
      </div>
    );
  }
}

export default AuthenticationFormWrapper;
