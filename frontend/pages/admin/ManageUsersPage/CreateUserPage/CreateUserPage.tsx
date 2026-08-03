import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { IApiError } from "interfaces/errors";
import { ITeam } from "interfaces/team";
import { IFormErrors } from "hooks/useFormValidation";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import usersAPI from "services/entities/users";
import invitesAPI from "services/entities/invites";

import BackButton from "components/BackButton";
import MainContent from "components/MainContent";
import { notify } from "components/ToastNotification";
import UserForm from "../components/UserForm";
import { IUserFormData, NewUserType } from "../components/UserForm/UserForm";

const baseClass = "create-user-page";

interface ICreateUserPageProps {
  router: InjectedRouter;
}

/**
 * Maps an API failure to inline field errors, or null when the failure isn't
 * field-specific and belongs in a toast instead.
 */
const getFieldErrors = (userErrors: {
  data: IApiError;
}): IFormErrors | null => {
  const reason = userErrors.data.errors?.[0]?.reason ?? "";

  if (reason.includes("already exists") || reason.includes("Duplicate")) {
    return { email: "Enter an email that isn't already in use" };
  }
  if (reason.includes("required criteria")) {
    return { password: "Enter a password that meets the requirements below" };
  }
  if (reason.includes("password too long")) {
    return { password: "Enter a password with 48 characters or fewer" };
  }
  return null;
};

const CreateUserPage = ({ router }: ICreateUserPageProps) => {
  const { config, currentUser, isPremiumTier } = useContext(AppContext);

  const [formErrors, setFormErrors] = useState<IFormErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { data: teams } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      enabled: !!isPremiumTier,
      select: (data: ILoadTeamsResponse) => data.teams,
    }
  );

  const handleSubmit = (formData: IUserFormData) => {
    setIsSubmitting(true);

    if (formData.newUserType === NewUserType.AdminInvited) {
      const requestData = {
        ...formData,
        invited_by: formData.currentUserId,
      };
      delete requestData.currentUserId;
      delete requestData.newUserType;
      delete requestData.password;
      invitesAPI
        .create(requestData)
        .then(() => {
          notify.success(`${formData.name} has been invited!`);
          router.push(PATHS.ADMIN_USERS);
        })
        .catch((userErrors: { data: IApiError }) => {
          const fieldErrors = getFieldErrors(userErrors);
          if (fieldErrors) {
            setFormErrors(fieldErrors);
          } else {
            notify.error("Could not create user. Please try again.", {
              response: userErrors,
            });
          }
        })
        .finally(() => {
          setIsSubmitting(false);
        });
    } else {
      const requestData = {
        ...formData,
      };
      delete requestData.currentUserId;
      delete requestData.newUserType;
      usersAPI
        .createUserWithoutInvitation(requestData)
        .then(() => {
          notify.success(`${requestData.name} has been created!`);
          router.push(PATHS.ADMIN_USERS);
        })
        .catch((userErrors: { data: IApiError }) => {
          const fieldErrors = getFieldErrors(userErrors);
          if (fieldErrors) {
            setFormErrors(fieldErrors);
          } else {
            notify.error("Could not create user. Please try again.", {
              response: userErrors,
            });
          }
        })
        .finally(() => {
          setIsSubmitting(false);
        });
    }
  };

  return (
    <MainContent className={baseClass}>
      <>
        <BackButton text="Back to users" path={PATHS.ADMIN_USERS} />
        <h1>New user</h1>
        <UserForm
          isNewUser
          isModifiedByGlobalAdmin
          onCancel={() => router.push(PATHS.ADMIN_USERS)}
          onSubmit={handleSubmit}
          availableTeams={teams || []}
          isPremiumTier={isPremiumTier || false}
          smtpConfigured={config?.smtp_settings?.configured || false}
          sesConfigured={config?.email?.backend === "ses" || false}
          canUseSso={config?.sso_settings?.enable_sso || false}
          currentUserId={currentUser?.id}
          serverErrors={formErrors}
          isUpdatingUsers={isSubmitting}
        />
      </>
    </MainContent>
  );
};

export default CreateUserPage;
