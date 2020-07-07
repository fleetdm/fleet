import React, { Component } from 'react';
import { connect } from 'react-redux';

import Icon from 'components/icons/Icon';

class LoginSuccessfulPage extends Component {
  render () {
    const baseClass = 'login-success';
    return (
      <div className={baseClass}>
        <Icon name="success-check" className={`${baseClass}__icon`} />
        <p className={`${baseClass}__text`}>Login successful</p>
        <p className={`${baseClass}__sub-text`}>Taking you to the Fleet UI...</p>
      </div>
    );
  }
}

const ConnectedComponent = connect()(LoginSuccessfulPage);
export default ConnectedComponent;
