import React, { useContext, useState, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { Params } from "react-router/lib/Router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { ICreateUserWithInvitationFormData } from "interfaces/user";
import paths from "router/paths";
import usersAPI from "services/entities/users";
import formatErrorResponse from "utilities/format_error_response";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
// @ts-ignore
import ConfirmInviteForm from "components/forms/ConfirmInviteForm";

interface IConfirmInvitePageProps {
  router: InjectedRouter; // v3
  location: any; // no type in react-router v3
  params: Params;
}

const baseClass = "confirm-invite-page";

const ConfirmInvitePage = ({
  router,
  location,
  params,
}: IConfirmInvitePageProps) => {
  const { email, name } = location.query;
  const { invite_token } = params;
  const inviteFormData = { email, invite_token, name };
  const { currentUser } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const [userErrors, setUserErrors] = useState<any>({});

  useEffect(() => {
    const { DASHBOARD } = paths;

    if (currentUser) {
      return router.push(DASHBOARD);
    }
  }, [currentUser]);

  const onSubmit = async (formData: ICreateUserWithInvitationFormData) => {
    const { create } = usersAPI;
    const { LOGIN } = paths;

    setUserErrors({});

    try {
      await create(formData);

      router.push(LOGIN);
      renderFlash(
        "success",
        "Registration successful! For security purposes, please log in."
      );
    } catch (error) {
      console.error(error);
      const errorsObject = formatErrorResponse(error);
      setUserErrors(errorsObject);
    }
  };

  return (
    <AuthenticationFormWrapper>
      <div className={`${baseClass}`}>
        <div className={`${baseClass}__lead-wrapper`}>
          <p className={`${baseClass}__lead-text`}>Welcome to Fleet</p>
          <p className={`${baseClass}__sub-lead-text`}>
            Before you get started, please take a moment to complete the
            following information.
          </p>
        </div>
        <ConfirmInviteForm
          className={`${baseClass}__form`}
          formData={inviteFormData}
          handleSubmit={onSubmit}
          serverErrors={userErrors}
        />
      </div>
    </AuthenticationFormWrapper>
  );
};

export default ConfirmInvitePage;
