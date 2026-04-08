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

import BackButton from "components/BackButton";
import MainContent from "components/MainContent";
import ApiUserForm from "../components/ApiUserForm";
import { IApiUserFormData } from "../components/ApiUserForm/ApiUserForm";
import ApiKeyDisplay from "../components/ApiKeyDisplay";

const baseClass = "create-api-user-page";

interface ICreateApiUserPageProps {
  router: InjectedRouter;
}

const CreateApiUserPage = ({ router }: ICreateApiUserPageProps) => {
  const { isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [formErrors, setFormErrors] = useState<IUserFormErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [apiKey, setApiKey] = useState<string | null>(null);
  const [createdUserName, setCreatedUserName] = useState("");

  const { data: teams } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      enabled: !!isPremiumTier,
      select: (data: ILoadTeamsResponse) => data.teams,
    }
  );

  const handleSubmit = (formData: IApiUserFormData) => {
    setIsSubmitting(true);
    setFormErrors({});

    usersAPI
      .createUserWithoutInvitation({
        name: formData.name,
        global_role: formData.global_role,
        teams: formData.teams,
        api_only: true,
      })
      .then((response) => {
        setCreatedUserName(formData.name);
        if (response.token) {
          setApiKey(response.token);
        } else {
          renderFlash("success", `${formData.name} has been created!`);
          router.push(PATHS.ADMIN_USERS);
        }
      })
      .catch((userErrors: { data: IApiError }) => {
        if (userErrors.data.errors[0].reason.includes("Duplicate")) {
          setFormErrors({
            name: "A user with this name already exists",
          });
        } else {
          renderFlash("error", "Could not create user. Please try again.");
        }
      })
      .finally(() => {
        setIsSubmitting(false);
      });
  };

  const handleDone = () => {
    renderFlash("success", `${createdUserName} has been created!`);
    router.push(PATHS.ADMIN_USERS);
  };

  return (
    <MainContent className={baseClass}>
      <>
        <BackButton text="Back to users" path={PATHS.ADMIN_USERS} />
        <h1>Add user</h1>
        {apiKey ? (
          <ApiKeyDisplay
            apiKey={apiKey}
            userName={createdUserName}
            onDone={handleDone}
          />
        ) : (
          <ApiUserForm
            isNewUser
            onCancel={() => router.push(PATHS.ADMIN_USERS)}
            onSubmit={handleSubmit}
            availableTeams={teams || []}
            formErrors={formErrors}
            isSubmitting={isSubmitting}
          />
        )}
      </>
    </MainContent>
  );
};

export default CreateApiUserPage;
