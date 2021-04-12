import React, { Component } from "react";
import { connect } from "react-redux";

class LoginSuccessfulPage extends Component {
  render() {
    const baseClass = "login-success";
    return (
      <div className={baseClass}>
        <p className={`${baseClass}__text`}>Login successful</p>
        <p className={`${baseClass}__sub-text`}>
          Taking you to the Fleet UI...
        </p>
      </div>
    );
  }
}

const ConnectedComponent = connect()(LoginSuccessfulPage);
export default ConnectedComponent;
