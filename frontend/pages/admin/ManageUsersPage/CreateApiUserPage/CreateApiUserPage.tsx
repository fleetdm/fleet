import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { ITeam } from "interfaces/team";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import usersAPI from "services/entities/users";

import BackButton from "components/BackButton";
import MainContent from "components/MainContent";
import PageDescription from "components/PageDescription";
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

    usersAPI
      .createApiOnlyUser({
        name: formData.name,
        global_role: formData.global_role,
        fleets: formData.fleets.map((f) => ({
          id: f.id,
          role: f.role ?? "observer",
        })),
        api_endpoints: formData.api_endpoints,
      })
      .then((response) => {
        setCreatedUserName(formData.name);
        if (response.token) {
          setApiKey(response.token);
        } else {
          renderFlash(
            "warning-filled",
            `${formData.name} has been created, but the API key could not be retrieved. Contact your administrator.`
          );
          router.push(PATHS.ADMIN_USERS);
        }
      })
      .catch(() => {
        renderFlash("error", "Could not create user. Please try again.");
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
      <BackButton text="Back to users" path={PATHS.ADMIN_USERS} />
      {apiKey ? (
        <ApiKeyDisplay
          newUserName={createdUserName}
          apiKey={apiKey}
          onDone={handleDone}
        />
      ) : (
        <>
          <div>
            <h1>New API-only user</h1>
            <PageDescription content="This user will have access to the Fleet API, but will not be able to log into the UI." />
          </div>
          <ApiUserForm
            isPremiumTier={isPremiumTier}
            onCancel={() => router.push(PATHS.ADMIN_USERS)}
            onSubmit={handleSubmit}
            availableTeams={teams || []}
            isSubmitting={isSubmitting}
          />
        </>
      )}
    </MainContent>
  );
};

export default CreateApiUserPage;
