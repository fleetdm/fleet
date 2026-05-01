import React, { useCallback, useContext, useState, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { Params } from "react-router/lib/Router";

import { AppContext } from "context/app";
import { RoutingContext } from "context/routing";
import paths from "router/paths";
import local from "utilities/local";
import authToken from "utilities/auth_token";
import configAPI from "services/entities/config";
import sessionsAPI from "services/entities/sessions";

import Button from "components/buttons/Button";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
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
  const [shouldFinishMFA, setShouldFinishMFA] = useState(
    !!local.getItem("auth_pending_mfa")
  );
  local.removeItem("auth_pending_mfa");

  const finishMFA = useCallback(async () => {
    const { DASHBOARD, RESET_PASSWORD, NO_ACCESS } = paths;

    try {
      const response = await sessionsAPI.finishMFA({ token: mfaToken });
      const { user, available_teams, token, token_expires_at } = response;

      const expiresAt = token_expires_at
        ? new Date(token_expires_at)
        : undefined;
      authToken.save(token, expiresAt);

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
  }, [
    config,
    mfaToken,
    redirectLocation,
    router,
    setAvailableTeams,
    setConfig,
    setCurrentTeam,
    setCurrentUser,
  ]);

  useEffect(() => {
    if (shouldFinishMFA) {
      finishMFA();
    }
  }, [shouldFinishMFA, finishMFA]);

  useEffect(() => {
    if (currentUser) {
      router.push(paths.DASHBOARD);
    }
    return undefined;
  }, [currentUser, router]);

  const onClickLoginButton = () => {
    router.push(paths.LOGIN);
  };

  const onClickFinishLoginButton = () => {
    setShouldFinishMFA(true);
  };

  if (!shouldFinishMFA) {
    return (
      <AuthenticationFormWrapper className={baseClass}>
        <Button onClick={onClickFinishLoginButton}>Log in</Button>
      </AuthenticationFormWrapper>
    );
  }

  if (isExpired) {
    return (
      <AuthenticationFormWrapper className={baseClass} header="Invalid token">
        <>
          <div className={`${baseClass}__description`}>
            <p>Log in again for a new link.</p>
          </div>
          <Button onClick={onClickLoginButton}>Back to login</Button>
        </>
      </AuthenticationFormWrapper>
    );
  }

  return (
    <AuthenticationFormWrapper className={baseClass}>
      <Spinner />
    </AuthenticationFormWrapper>
  );
};

export default MfaPage;
