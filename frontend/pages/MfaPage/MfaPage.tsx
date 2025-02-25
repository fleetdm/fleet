import React, { useContext, useState, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { Params } from "react-router/lib/Router";

import { AppContext } from "context/app";
import { RoutingContext } from "context/routing";
import paths from "router/paths";
import local from "utilities/local";
import configAPI from "services/entities/config";
import sessionsAPI from "services/entities/sessions";

import Button from "components/buttons/Button";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import StackedWhiteBoxes from "components/StackedWhiteBoxes";
import Spinner from "components/Spinner";

interface IMfaPage {
  router: InjectedRouter; // v3
  params: Params;
}

const baseClass = "mfa-page";

const MfaPage = ({ router, params }: IMfaPage) => {
  const { token: mfaToken } = params;
  const {
    config,
    currentUser,
    setAvailableTeams,
    setConfig,
    setCurrentUser,
    setCurrentTeam,
  } = useContext(AppContext);
  const { redirectLocation } = useContext(RoutingContext);
  const [isExpired, setIsExpired] = useState(false);

  const finishMFA = async () => {
    const { DASHBOARD, RESET_PASSWORD, NO_ACCESS } = paths;

    try {
      const response = await sessionsAPI.finishMFA({ token: mfaToken });
      const { user, available_teams, token } = response;

      local.setItem("auth_token", token);

      setCurrentUser(user);
      setAvailableTeams(user, available_teams);
      setCurrentTeam(undefined);

      if (!user.global_role && user.teams.length === 0) {
        router.push(NO_ACCESS);
        return;
      }
      // Redirect to password reset page if user is forced to reset password.
      // Any other requests will fail.
      else if (user.force_password_reset) {
        router.push(RESET_PASSWORD);
        return;
      } else if (config) {
        router.push(redirectLocation || DASHBOARD);
        return;
      }

      configAPI.loadAll().then((configResponse) => {
        setConfig(configResponse);
        router.push(redirectLocation || DASHBOARD);
      });
    } catch (response) {
      setIsExpired(true);
    }
  };

  useEffect(() => {
    finishMFA();
  });

  useEffect(() => {
    if (currentUser) {
      return router.push(paths.DASHBOARD);
    }
  }, [currentUser, router]);

  const onClickLoginButton = () => {
    router.push(paths.LOGIN);
  };

  if (isExpired) {
    return (
      <AuthenticationFormWrapper>
        <StackedWhiteBoxes className={baseClass}>
          <>
            <p>
              <b>That link is expired.</b>
            </p>
            <p>Log in again for a new link.</p>
            <Button variant="brand" onClick={onClickLoginButton}>
              Back to login
            </Button>
          </>
        </StackedWhiteBoxes>
      </AuthenticationFormWrapper>
    );
  }

  return (
    <AuthenticationFormWrapper>
      <div className={baseClass}>
        <Spinner />
      </div>
    </AuthenticationFormWrapper>
  );
};

export default MfaPage;
