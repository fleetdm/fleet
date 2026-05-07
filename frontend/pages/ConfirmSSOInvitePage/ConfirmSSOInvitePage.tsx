import React, { useCallback, useContext, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { Params } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import paths from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import usersAPI from "services/entities/users";
import sessionsAPI from "services/entities/sessions";
import inviteAPI, { IValidateInviteResp } from "services/entities/invites";
import { IInvite } from "interfaces/invite";
import { getErrorReason } from "interfaces/errors";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import Spinner from "components/Spinner";
import ConfirmSSOInviteForm from "components/forms/ConfirmSSOInviteForm";

interface IConfirmSSOInvitePageProps {
  params: Params;
  router: InjectedRouter;
}

const baseClass = "confirm-invite-page";

const ConfirmSSOInvitePage = ({
  params,
  router,
}: IConfirmSSOInvitePageProps) => {
  const { invite_token } = params;
  const { currentUser } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  useEffect(() => {
    if (currentUser) {
      router.push(paths.DASHBOARD);
    }
  }, [currentUser, router]);

  const {
    data: validInvite,
    error: validateInviteError,
    isLoading: isVerifyingInvite,
  } = useQuery<IValidateInviteResp, AxiosError, IInvite>(
    ["invite", invite_token],
    () => inviteAPI.verify(invite_token),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (resp: IValidateInviteResp) => resp.invite,
    }
  );

  const onSubmit = useCallback(
    async (name: string) => {
      // The form is only rendered once the invite has been verified, so
      // validInvite is always defined here. The early return tightens
      // types and guards against future drift.
      if (!validInvite) return;

      try {
        await usersAPI.create({
          email: validInvite.email,
          invite_token,
          name,
          sso_invite: true,
        });
        const { url } = await sessionsAPI.initializeSSO(paths.DASHBOARD);
        window.location.href = url;
      } catch (error) {
        renderFlash("error", getErrorReason(error));
      }
    },
    [invite_token, renderFlash, validInvite]
  );

  const renderContent = () => {
    if (isVerifyingInvite) {
      return <Spinner />;
    }

    if (validateInviteError || !validInvite) {
      return (
        <p className={`${baseClass}__description`}>
          This invite token is invalid. Please confirm your invite link.
        </p>
      );
    }

    return (
      <>
        <p className={`${baseClass}__description`}>
          Please provide your name to get started.
        </p>
        <ConfirmSSOInviteForm
          defaultName={validInvite.name}
          email={validInvite.email}
          handleSubmit={onSubmit}
        />
      </>
    );
  };

  return (
    <AuthenticationFormWrapper
      header={validateInviteError ? "Invalid invite token" : "Welcome to Fleet"}
      className={baseClass}
    >
      {renderContent()}
    </AuthenticationFormWrapper>
  );
};

export default ConfirmSSOInvitePage;
