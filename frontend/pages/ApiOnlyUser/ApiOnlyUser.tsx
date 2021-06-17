import React from "react";
import { useDispatch, useSelector, useStore } from "react-redux";
import { push } from "react-router-redux";
// @ts-ignore
import { logoutUser } from "redux/nodes/auth/actions";
import Button from "components/buttons/Button";
import paths from "router/paths";
import { IUser } from "interfaces/user";
// @ts-ignore
import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";

const baseClass = "api-only-user";
interface IRootState {
  auth: {
    user: IUser;
  };
}

const ApiOnlyUser = (): JSX.Element => {
  const dispatch = useDispatch();
  const { LOGIN, HOME } = paths;
  const handleClick = (event: any) => dispatch(logoutUser());

  // These are showing up empty. Need to be able to get state loaded in from redux store.
  // const auth = useSelector((state: any) => state.auth);
  // const app = useSelector((state: any) => state.app);
  // console.log("state.auth:", auth);
  // console.log("state.app:", app);

  const user = useSelector((state: IRootState) => state.auth.user);

  if (!user) {
    dispatch(push(LOGIN));
  } else if (user && !user.api_only) {
    dispatch(push(HOME));
  }

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
