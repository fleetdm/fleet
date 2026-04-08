import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { IApiError } from "interfaces/errors";
import { ITeam } from "interfaces/team";
import { IUser, IUserFormErrors } from "interfaces/user";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import usersAPI from "services/entities/users";

import BackButton from "components/BackButton";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import UserForm from "../components/UserForm";
import { IUserFormData } from "../components/UserForm/UserForm";
import ApiUserForm from "../components/ApiUserForm";
import { IApiUserFormData } from "../components/ApiUserForm/ApiUserForm";

const baseClass = "edit-user-page";

interface IEditUserPageProps {
  router: InjectedRouter;
  params: { user_id: string };
}

const EditUserPage = ({ router, params }: IEditUserPageProps) => {
  const userId = parseInt(params.user_id, 10);
  const { config, isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [formErrors, setFormErrors] = useState<IUserFormErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { data: user, isLoading: isLoadingUser, error: userError } = useQuery<
    IUser,
    Error
  >(["user", userId], () => usersAPI.getUserById(userId));

  const { data: teams } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      enabled: !!isPremiumTier,
      select: (data: ILoadTeamsResponse) => data.teams,
    }
  );

  const handleHumanUserSubmit = (formData: IUserFormData) => {
    if (!user) return;
    setIsSubmitting(true);
    setFormErrors({});

    const requestData: Record<string, unknown> = {};
    if (formData.name !== user.name) requestData.name = formData.name;
    if (formData.email !== user.email) requestData.email = formData.email;
    if (formData.sso_enabled !== user.sso_enabled)
      requestData.sso_enabled = formData.sso_enabled;
    if (formData.mfa_enabled !== user.mfa_enabled)
      requestData.mfa_enabled = formData.mfa_enabled;
    if (formData.global_role !== user.global_role)
      requestData.global_role = formData.global_role;
    if (formData.teams) requestData.teams = formData.teams;
    if (formData.new_password) requestData.new_password = formData.new_password;

    let successMessage = `Successfully edited ${formData.name}`;
    if (user.email !== formData.email) {
      successMessage += `. A confirmation email was sent to ${formData.email}.`;
    }

    usersAPI
      .update(userId, requestData)
      .then(() => {
        renderFlash("success", successMessage);
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
        } else {
          renderFlash(
            "error",
            `Could not edit ${user.name}. Please try again.`
          );
        }
      })
      .finally(() => {
        setIsSubmitting(false);
      });
  };

  const handleApiUserSubmit = (formData: IApiUserFormData) => {
    if (!user) return;
    setIsSubmitting(true);
    setFormErrors({});

    usersAPI
      .update(userId, {
        name: formData.name,
        global_role: formData.global_role,
        teams: formData.teams,
      })
      .then(() => {
        renderFlash("success", `Successfully edited ${formData.name}.`);
        router.push(PATHS.ADMIN_USERS);
      })
      .catch(() => {
        renderFlash("error", `Could not edit ${user.name}. Please try again.`);
      })
      .finally(() => {
        setIsSubmitting(false);
      });
  };

  if (isLoadingUser) {
    return (
      <MainContent className={baseClass}>
        <Spinner />
      </MainContent>
    );
  }

  if (userError || !user) {
    return (
      <MainContent className={baseClass}>
        <>
          <BackButton text="Back to users" path={PATHS.ADMIN_USERS} />
          <DataError />
        </>
      </MainContent>
    );
  }

  return (
    <MainContent className={baseClass}>
      <>
        <BackButton text="Back to users" path={PATHS.ADMIN_USERS} />
        <h1>Edit user</h1>
        {user.api_only ? (
          <ApiUserForm
            onCancel={() => router.push(PATHS.ADMIN_USERS)}
            onSubmit={handleApiUserSubmit}
            availableTeams={teams || []}
            defaultName={user.name}
            defaultGlobalRole={user.global_role}
            defaultTeams={user.teams}
            formErrors={formErrors}
            isSubmitting={isSubmitting}
          />
        ) : (
          <UserForm
            onCancel={() => router.push(PATHS.ADMIN_USERS)}
            onSubmit={handleHumanUserSubmit}
            availableTeams={teams || []}
            isPremiumTier={isPremiumTier || false}
            smtpConfigured={config?.smtp_settings?.configured || false}
            sesConfigured={config?.email?.backend === "ses" || false}
            canUseSso={config?.sso_settings?.enable_sso || false}
            isSsoEnabled={user.sso_enabled}
            isMfaEnabled={user.mfa_enabled}
            isApiOnly={user.api_only}
            isModifiedByGlobalAdmin
            defaultName={user.name}
            defaultEmail={user.email}
            defaultGlobalRole={user.global_role}
            defaultTeams={user.teams}
            ancestorErrors={formErrors}
            isUpdatingUsers={isSubmitting}
          />
        )}
      </>
    </MainContent>
  );
};

export default EditUserPage;
