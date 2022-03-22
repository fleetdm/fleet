import React, { Component } from "react";

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

export default LoginSuccessfulPage;
