import React, { useEffect } from "react";
import { InjectedRouter } from "react-router";

import paths from "router/paths";
import usersAPI from "services/entities/users";

import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";

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
    <AuthenticationFormWrapper header="Access denied" className={baseClass}>
      <>
        <div>
          <p>
            You attempted to access Fleet with an{" "}
            <CustomLink
              text="API only user"
              newTab
              url="https://fleetdm.com/docs/using-fleet/fleetctl-cli#using-fleetctl-with-an-api-only-user"
            />
            .
          </p>
          <p className={`${baseClass}__sub-lead-text`}>
            This user doesn&apos;t have access to the Fleet UI.
          </p>
        </div>
        <Button onClick={handleClick} className={`${baseClass}__login-button`}>
          Back to login
        </Button>
      </>
    </AuthenticationFormWrapper>
  );
};

export default ApiOnlyUser;
