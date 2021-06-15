import React, { Component } from "react";

import Button from "components/buttons/Button";
import PATHS from "router/paths";

const baseClass = "api-only-user";

class ApiOnlyUser extends Component {
  render() {
    return (
      <div className={`${baseClass}`}>
        <div className={`${baseClass}__lead-wrapper`}>
          <p className={`${baseClass}__lead-text`}>
            You attempted to access Fleet with an API only user.
          </p>
          <p className={`${baseClass}__sub-lead-text`}>
            This user doesn't have access to the Fleet UI.
          </p>
        </div>
        <Button
          TO={PATHS.LOGIN}
          variant="brand"
          className={`${baseClass}__login-button`}
        >
          Back to login
        </Button>
      </div>
    );
  }
}

export default ApiOnlyUser;
