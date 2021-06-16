import React, { Component } from "react";

import Button from "components/buttons/Button";
import PATHS from "router/paths";
import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";

const baseClass = "api-only-user";

class ApiOnlyUser extends Component {
  render() {
    return (
      <div className="api-only-user">
        <img alt="Fleet" src={fleetLogoText} className={`${baseClass}__logo`} />
        <div className={`${baseClass}__wrap`}>
          <div className={`${baseClass}__lead-wrapper`}>
            <p className={`${baseClass}__lead-text`}>
              You attempted to access Fleet with an API only user.
            </p>
            <p className={`${baseClass}__sub-lead-text`}>
              This user doesn't have access to the Fleet UI.
            </p>
          </div>
          <div className="login-button-wrap">
            <Button
              TO={PATHS.LOGIN}
              variant="brand"
              className={`${baseClass}__login-button`}
            >
              Back to login
            </Button>
          </div>
        </div>
      </div>
    );
  }
}

export default ApiOnlyUser;
