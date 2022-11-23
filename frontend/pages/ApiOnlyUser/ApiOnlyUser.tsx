import React, { useEffect } from "react";
import { InjectedRouter } from "react-router";

import paths from "router/paths";
import usersAPI from "services/entities/users";

import Button from "components/buttons/Button";
// @ts-ignore
import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";

interface IApiOnlyUserProps {
  router: InjectedRouter;
}

const baseClass = "api-only-user";

const ApiOnlyUser = ({ router }: IApiOnlyUserProps): JSX.Element => {
  const { LOGIN, DASHBOARD, LOGOUT } = paths;
  const handleClick = () => router.push(LOGOUT);

  useEffect(() => {
    const fetchCurrentUser = async () => {
      try {
        const { user } = await usersAPI.me();

        if (!user) {
          router.push(LOGIN);
        } else if (!user?.api_only) {
          router.push(DASHBOARD);
        }
      } catch (response) {
        console.error(response);
        return false;
      }
    };

    fetchCurrentUser();
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
