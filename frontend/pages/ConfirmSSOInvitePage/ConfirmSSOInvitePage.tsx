import React, { useState, useEffect, useContext } from "react";
import { InjectedRouter } from "react-router";
import { Params } from "react-router/lib/Router";

import paths from "router/paths";
import { AppContext } from "context/app";
import usersAPI from "services/entities/users";
import sessionsAPI from "services/entities/sessions";
import formatErrorResponse from "utilities/format_error_response";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
// @ts-ignore
import ConfirmSSOInviteForm from "components/forms/ConfirmSSOInviteForm";

interface IConfirmSSOInvitePageProps {
  location: any; // no type in react-router v3
  params: Params;
  router: InjectedRouter;
}

const baseClass = "confirm-invite-page";

const ConfirmSSOInvitePage = ({
  location,
  params,
  router,
}: IConfirmSSOInvitePageProps) => {
  const { email, name } = location.query;
  const { invite_token } = params;
  const inviteFormData = { email, invite_token, name };
  const { currentUser } = useContext(AppContext);
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  useEffect(() => {
    const { DASHBOARD } = paths;

    if (currentUser) {
      return router.push(DASHBOARD);
    }
  }, [currentUser]);

  const onSubmit = async (formData: any) => {
    const { DASHBOARD } = paths;

    formData.sso_invite = true;

    try {
      await usersAPI.create(formData);
      const { url } = await sessionsAPI.initializeSSO(DASHBOARD);
      window.location.href = url;
    } catch (response) {
      const errorObject = formatErrorResponse(response);
      setErrors(errorObject);
      return false;
    }
  };

  return (
    <AuthenticationFormWrapper className={baseClass} header="Welcome to Fleet">
      <>
        <p className={`${baseClass}__description`}>
          Please provide your name to get started.
        </p>
        <ConfirmSSOInviteForm
          className={`${baseClass}__form`}
          formData={inviteFormData}
          handleSubmit={onSubmit}
          serverErrors={errors}
        />
      </>
    </AuthenticationFormWrapper>
  );
};

export default ConfirmSSOInvitePage;
