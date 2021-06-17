import React from "react";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";

import Button from "components/buttons/Button";
import paths from "router/paths";
// @ts-ignore
import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";

const baseClass = "api-only-user";

const ApiOnlyUser = (): JSX.Element | null => {
  const dispatch = useDispatch();
  const { LOGIN } = paths;
  const handleClick = (event: any) => dispatch(push(LOGIN));

  return (
    <div className="api-only-user">
      <img alt="Fleet" src={fleetLogoText} className={`${baseClass}__logo`} />
      <div className={`${baseClass}__wrap`}>
        <div className={`${baseClass}__lead-wrapper`}>
          <p className={`${baseClass}__lead-text`}>
            You attempted to access Fleet with an API only user.
          </p>
          <p className={`${baseClass}__sub-lead-text`}>
            This user doesn&apos;t have access to the Fleet UI.
          </p>
        </div>
        <div className="login-button-wrap">
          <Button
            onClick={handleClick}
            variant="brand"
            className={`${baseClass}__login-button`}
          >
            Back to login
          </Button>
        </div>
      </div>
    </div>
  );
};

export default ApiOnlyUser;
