import React, { Component } from "react";
import PropTypes from "prop-types";

import { hideBackgroundImage } from "redux/nodes/app/actions";
import { ssoSettings } from "redux/nodes/auth/actions";

import Button from "components/buttons/Button";
import PATHS from "router/paths";

const baseClass = "api-only-user";

class ApiOnlyUser extends Component {
  // static propTypes = {
  //   dispatch: PropTypes.func,
  // };

  // componentWillMount() {
  //   const { dispatch } = this.props;

  //   dispatch(ssoSettings()).catch(() => false);

  //   dispatch(hideBackgroundImage);
  // }

  // componentWillUnmount() {
  //   const { dispatch } = this.props;

  //   dispatch(hideBackgroundImage);
  // }

  render() {
    return (
      <div className="api-only-user">
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
