import React, { Component } from "react";
import PropTypes from "prop-types";

import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";

const baseClass = "auth-form-wrapper";

class AuthenticationFormWrapper extends Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render() {
    const { children } = this.props;

    return (
      <div className={baseClass}>
        <img alt="Fleet" src={fleetLogoText} className={`${baseClass}__logo`} />
        {children}
      </div>
    );
  }
}

export default AuthenticationFormWrapper;
