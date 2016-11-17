import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';

class LoginSuccessfulPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func.isRequired,
  };

  render () {
    const baseClass = 'login-success';
    return (
      <div className={baseClass}>
        <i className={`${baseClass}__icon kolidecon kolidecon-success-check`} />
        <p className={`${baseClass}__text`}>Login successful</p>
        <p className={`${baseClass}__sub-text`}>hold on to your butts...</p>
      </div>
    );
  }
}

const ConnectedComponent = connect()(LoginSuccessfulPage);
export default ConnectedComponent;
