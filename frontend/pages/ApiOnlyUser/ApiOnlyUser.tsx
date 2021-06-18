import React, { useEffect } from "react";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";
// @ts-ignore
import { fetchCurrentUser, logoutUser } from "redux/nodes/auth/actions";
import Button from "components/buttons/Button";
import paths from "router/paths";
// @ts-ignore
import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";

const baseClass = "api-only-user";

const ApiOnlyUser = (): JSX.Element => {
  const dispatch = useDispatch();
  const { LOGIN, HOME } = paths;
  const handleClick = (event: any) => dispatch(logoutUser());

  useEffect(() => {
    dispatch(fetchCurrentUser()).then((user: any) => {
      if (!user) {
        dispatch(push(LOGIN));
      } else if (user && !user.payload.user.api_only) {
        dispatch(push(HOME));
      }
    });
  }, []);

  return (
    <div className={baseClass}>
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
