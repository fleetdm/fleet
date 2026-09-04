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
import { getUserFieldErrors } from "../helpers/userManagementHelpers";

const baseClass = "create-user-page";

interface ICreateUserPageProps {
  router: InjectedRouter;
}

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
      return invitesAPI
        .create(requestData)
        .then(() => {
          notify.success(`${formData.name} has been invited!`);
          router.push(PATHS.ADMIN_USERS);
        })
        .catch((userErrors: { data: IApiError }) => {
          const fieldErrors = getUserFieldErrors(userErrors);
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

    const requestData = {
      ...formData,
    };
    delete requestData.currentUserId;
    delete requestData.newUserType;
    return usersAPI
      .createUserWithoutInvitation(requestData)
      .then(() => {
        notify.success(`${requestData.name} has been created!`);
        router.push(PATHS.ADMIN_USERS);
      })
      .catch((userErrors: { data: IApiError }) => {
        const fieldErrors = getUserFieldErrors(userErrors);
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
