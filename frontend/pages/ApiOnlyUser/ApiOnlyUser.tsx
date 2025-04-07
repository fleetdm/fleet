import React, { useEffect } from "react";
import { InjectedRouter } from "react-router";

import paths from "router/paths";
import usersAPI from "services/entities/users";

import Button from "components/buttons/Button";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import StackedWhiteBoxes from "components/StackedWhiteBoxes";

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
      <AuthenticationFormWrapper>
        <StackedWhiteBoxes router={router}>
          <>
            <p>You attempted to access Fleet with an API only user.</p>
            <p className={`${baseClass}__sub-lead-text`}>
              This user doesn&apos;t have access to the Fleet UI.
            </p>
            <Button
              onClick={handleClick}
              variant="brand"
              className={`${baseClass}__login-button`}
            >
              Back to login
            </Button>
          </>
        </StackedWhiteBoxes>
      </AuthenticationFormWrapper>
    </div>
  );
};

export default ApiOnlyUser;
