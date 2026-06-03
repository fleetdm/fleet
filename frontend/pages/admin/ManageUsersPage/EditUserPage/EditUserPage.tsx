import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery, useQueryClient } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { IApiError } from "interfaces/errors";
import { ITeam } from "interfaces/team";
import { IInvite, IEditInviteFormData } from "interfaces/invite";
import { IUser, IUserFormErrors } from "interfaces/user";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import usersAPI from "services/entities/users";
import invitesAPI from "services/entities/invites";

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
  location: {
    query?: { type?: string };
  };
}

const EditUserPage = ({ router, params, location }: IEditUserPageProps) => {
  const entityId = parseInt(params.user_id, 10);
  const isInvite = location.query?.type === "invite";
  const { config, isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const queryClient = useQueryClient();
  const [formErrors, setFormErrors] = useState<IUserFormErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Fetch user (when not an invite)
  const { data: user, isLoading: isLoadingUser, error: userError } = useQuery<
    IUser,
    Error
  >(["user", entityId], () => usersAPI.getUserById(entityId), {
    enabled: !isInvite,
  });

  // Fetch invite (when editing an invite)
  const {
    data: invite,
    isLoading: isLoadingInvite,
    error: inviteError,
  } = useQuery<IInvite[], Error, IInvite | undefined>(
    ["invites", entityId],
    () => invitesAPI.loadAll({ globalFilter: "" }),
    {
      enabled: isInvite,
      select: (invites) => invites.find((i) => i.id === entityId),
    }
  );

  const { data: teams, isLoading: isLoadingTeams } = useQuery<
    ILoadTeamsResponse,
    Error,
    ITeam[]
  >(["teams"], () => teamsAPI.loadAll(), {
    enabled: !!isPremiumTier,
    select: (data: ILoadTeamsResponse) => data.teams,
  });

  const isLoading =
    (isInvite ? isLoadingInvite : isLoadingUser) ||
    (isPremiumTier && isLoadingTeams);
  const hasError = isInvite ? !!inviteError : !!userError;
  const entityData = isInvite ? invite : user;

  const handleHumanUserSubmit = (formData: IUserFormData) => {
    if (!entityData) return;
    setIsSubmitting(true);
    setFormErrors({});

    if (isInvite) {
      invitesAPI
        .update(entityId, (formData as unknown) as IEditInviteFormData)
        .then(() => {
          let msg = `Successfully edited ${formData.name}`;
          if (entityData.email !== formData.email) {
            msg += `. A confirmation email was sent to ${formData.email}.`;
          }
          renderFlash("success", msg);
          router.push(PATHS.ADMIN_USERS);
        })
        .catch((inviteErrors: { data: IApiError }) => {
          if (inviteErrors.data.errors[0].reason.includes("already exists")) {
            setFormErrors({
              email: "A user with this email address already exists",
            });
          } else {
            renderFlash(
              "error",
              `Could not edit ${entityData.name}. Please try again.`
            );
          }
        })
        .finally(() => {
          setIsSubmitting(false);
        });
      return;
    }

    // Do not update password to empty string
    if (formData.new_password === "") {
      formData.new_password = null;
    }

    // Editing a regular user
    const requestData: Record<string, unknown> = {};
    if (formData.name !== entityData.name) requestData.name = formData.name;
    if (formData.email !== entityData.email) requestData.email = formData.email;
    if (formData.sso_enabled !== (entityData as IUser).sso_enabled)
      requestData.sso_enabled = formData.sso_enabled;
    if (formData.mfa_enabled !== (entityData as IUser).mfa_enabled)
      requestData.mfa_enabled = formData.mfa_enabled;
    if (formData.global_role !== entityData.global_role)
      requestData.global_role = formData.global_role;
    if (formData.teams && formData.teams.length > 0)
      requestData.teams = formData.teams;
    if (formData.new_password) requestData.new_password = formData.new_password;

    let successMessage = `Successfully edited ${formData.name}`;
    if (entityData.email !== formData.email) {
      successMessage += `. A confirmation email was sent to ${formData.email}.`;
    }

    usersAPI
      .update(entityId, requestData)
      .then(() => {
        queryClient.invalidateQueries(["user", entityId]);
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
            `Could not edit ${entityData.name}. Please try again.`
          );
        }
      })
      .finally(() => {
        setIsSubmitting(false);
      });
  };

  const handleApiUserSubmit = (formData: IApiUserFormData) => {
    if (!entityData) return;
    setIsSubmitting(true);
    setFormErrors({});

    usersAPI
      .updateApiOnlyUser(entityId, {
        name: formData.name,
        global_role: formData.global_role,
        fleets: formData.fleets.map((f) => ({
          id: f.id,
          role: f.role ?? "observer",
        })),
        api_endpoints: formData.api_endpoints,
      })
      .then(() => {
        queryClient.invalidateQueries(["user", entityId]);
        renderFlash("success", `Successfully edited ${formData.name}.`);
        router.push(PATHS.ADMIN_USERS);
      })
      .catch(() => {
        renderFlash(
          "error",
          `Could not edit ${entityData.name}. Please try again.`
        );
      })
      .finally(() => {
        setIsSubmitting(false);
      });
  };

  if (isLoading) {
    return (
      <MainContent className={baseClass}>
        <Spinner />
      </MainContent>
    );
  }

  if (hasError || !entityData) {
    return (
      <MainContent className={baseClass}>
        <BackButton text="Back to users" path={PATHS.ADMIN_USERS} />
        <DataError />
      </MainContent>
    );
  }

  const showApiForm = !isInvite && (entityData as IUser).api_only;

  return (
    <MainContent className={baseClass}>
      <BackButton text="Back to users" path={PATHS.ADMIN_USERS} />
      {showApiForm ? (
        <>
          <h1>Edit API-only user</h1>
          <ApiUserForm
            onCancel={() => router.push(PATHS.ADMIN_USERS)}
            onSubmit={handleApiUserSubmit}
            availableTeams={teams || []}
            defaultData={{
              name: entityData.name,
              global_role: entityData.global_role,
              fleets: entityData.teams,
              api_endpoints: (entityData as IUser).api_endpoints,
            }}
            isSubmitting={isSubmitting}
            isPremiumTier={isPremiumTier}
          />
        </>
      ) : (
        <>
          <h1>Edit user</h1>
          <UserForm
            onCancel={() => router.push(PATHS.ADMIN_USERS)}
            onSubmit={handleHumanUserSubmit}
            availableTeams={teams || []}
            isPremiumTier={isPremiumTier || false}
            smtpConfigured={config?.smtp_settings?.configured || false}
            sesConfigured={config?.email?.backend === "ses" || false}
            canUseSso={config?.sso_settings?.enable_sso || false}
            isSsoEnabled={entityData?.sso_enabled}
            isMfaEnabled={(entityData as IUser).mfa_enabled}
            isApiOnly={false}
            isInvitePending={isInvite}
            isModifiedByGlobalAdmin
            defaultName={entityData?.name}
            defaultEmail={entityData?.email}
            defaultGlobalRole={entityData?.global_role}
            defaultTeams={entityData?.teams}
            ancestorErrors={formErrors}
            isUpdatingUsers={isSubmitting}
          />
        </>
      )}
    </MainContent>
  );
};

export default EditUserPage;
