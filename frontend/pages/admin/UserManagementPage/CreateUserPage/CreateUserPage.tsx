import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { IApiError } from "interfaces/errors";
import { ITeam } from "interfaces/team";
import { IUserFormErrors } from "interfaces/user";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import usersAPI from "services/entities/users";
import invitesAPI from "services/entities/invites";

import BackButton from "components/BackButton";
import MainContent from "components/MainContent";
import UserForm from "../components/UserForm";
import { IUserFormData, NewUserType } from "../components/UserForm/UserForm";

const baseClass = "create-user-page";

interface ICreateUserPageProps {
  router: InjectedRouter;
}

const CreateUserPage = ({ router }: ICreateUserPageProps) => {
  const { config, currentUser, isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [formErrors, setFormErrors] = useState<IUserFormErrors>({});
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
          renderFlash("success", `${formData.name} has been invited!`);
          router.push(PATHS.ADMIN_USERS);
        })
        .catch((userErrors: { data: IApiError }) => {
          if (userErrors.data.errors[0].reason.includes("already exists")) {
            setFormErrors({
              email: "A user with this email address already exists",
            });
          } else if (
            userErrors.data.errors[0].reason.includes("required criteria")
          ) {
            setFormErrors({
              password: "Password must meet the criteria below",
            });
          } else if (
            userErrors.data.errors?.[0].reason.includes("password too long")
          ) {
            setFormErrors({
              password: "Password is over the character limit.",
            });
          } else {
            renderFlash("error", "Could not create user. Please try again.");
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
          renderFlash("success", `${requestData.name} has been created!`);
          router.push(PATHS.ADMIN_USERS);
        })
        .catch((userErrors: { data: IApiError }) => {
          if (userErrors.data.errors[0].reason.includes("Duplicate")) {
            setFormErrors({
              email: "A user with this email address already exists",
            });
          } else if (
            userErrors.data.errors[0].reason.includes("required criteria")
          ) {
            setFormErrors({
              password: "Password must meet the criteria below",
            });
          } else if (
            userErrors.data.errors?.[0].reason.includes("password too long")
          ) {
            setFormErrors({
              password: "Password is over the character limit.",
            });
          } else {
            renderFlash("error", "Could not create user. Please try again.");
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
          ancestorErrors={formErrors}
          isUpdatingUsers={isSubmitting}
        />
      </>
    </MainContent>
  );
};

export default CreateUserPage;
