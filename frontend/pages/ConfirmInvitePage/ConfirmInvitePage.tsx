import React, { useCallback, useContext } from "react";
import { InjectedRouter } from "react-router";
import { Params } from "react-router/lib/Router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { ICreateUserWithInvitationFormData } from "interfaces/user";
import paths from "router/paths";
import usersAPI from "services/entities/users";
import inviteAPI, { IValidateInviteResp } from "services/entities/invites";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import Spinner from "components/Spinner";
import { useQuery } from "react-query";
import { IInvite } from "interfaces/invite";
import StackedWhiteBoxes from "components/StackedWhiteBoxes";
import ConfirmInviteForm from "components/forms/ConfirmInviteForm";
import { IConfirmInviteFormData } from "components/forms/ConfirmInviteForm/ConfirmInviteForm";
import { getErrorReason } from "interfaces/errors";

interface IConfirmInvitePageProps {
  router: InjectedRouter; // v3
  params: Params;
}

const baseClass = "confirm-invite-page";

const ConfirmInvitePage = ({ router, params }: IConfirmInvitePageProps) => {
  const { currentUser } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const { invite_token } = params;

  const {
    data: validInvite,
    error: validateInviteError,
    isLoading: isVerifyingInvite,
  } = useQuery<IValidateInviteResp, Error, IInvite>(
    "invite",
    () => inviteAPI.verify(invite_token),
    { select: (resp: IValidateInviteResp) => resp.invite }
  );

  const onSubmit = useCallback(
    async (formData: IConfirmInviteFormData) => {
      const dataForAPI: ICreateUserWithInvitationFormData = {
        email: validInvite?.email || "",
        invite_token,
        name: formData.name,
        password: formData.password,
        password_confirmation: formData.password_confirmation,
      };

      try {
        await usersAPI.create(dataForAPI);
        router.push(paths.LOGIN);
        renderFlash(
          "success",
          "Registration successful! For security purposes, please log in."
        );
      } catch (error) {
        const reason = getErrorReason(error);
        console.error(reason);
        renderFlash("error", reason);
      }
    },
    [invite_token, renderFlash, router, validInvite?.email]
  );

  if (currentUser) {
    router.push(paths.DASHBOARD);
    // return for router typechecking
    return <></>;
  }

  const renderContent = () => {
    if (isVerifyingInvite) {
      return <Spinner />;
    }

    // error is how API communicates an invalid invite
    if (validateInviteError) {
      return (
        <StackedWhiteBoxes className={baseClass}>
          <>
            <p>
              <b>That invite is invalid.</b>
            </p>
            <p>Please confirm your invite link.</p>
          </>
        </StackedWhiteBoxes>
      );
    }
    // valid - return form pre-filled with data from api response
    return (
      <div className={`${baseClass}`}>
        <div className={`${baseClass}__lead-wrapper`}>
          <p className={`${baseClass}__lead-text`}>Welcome to Fleet</p>
          <p className={`${baseClass}__sub-lead-text`}>
            Before you get started, please take a moment to complete the
            following information.
          </p>
        </div>
        <ConfirmInviteForm
          defaultFormData={{
            // at this point we will have a valid invite per error check above
            name: validInvite?.name,
          }}
          handleSubmit={onSubmit}
        />
      </div>
    );
  };

  return (
    <AuthenticationFormWrapper>{renderContent()}</AuthenticationFormWrapper>
  );
};

export default ConfirmInvitePage;
