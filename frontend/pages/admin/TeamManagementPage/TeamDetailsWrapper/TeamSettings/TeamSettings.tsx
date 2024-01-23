import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import useTeamIdParam from "hooks/useTeamIdParam";
import { ITeamConfig } from "interfaces/team";
import { ITeamSubnavProps } from "interfaces/team_subnav";
import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import configAPI from "services/entities/config";
import { IConfig } from "interfaces/config";
import { NotificationContext } from "context/notification";
import { IApiError } from "interfaces/errors";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import TeamHostExpiryToggle from "./components/TeamHostExpiryToggle";

const baseClass = "team-settings";

const HOST_EXPIRY_ERROR_TEXT = "Host expiry window must be a positive number.";

const TeamSettings = ({ location, router }: ITeamSubnavProps) => {
  const [UITeamHostExpiryEnabled, setUITeamHostExpiryEnabled] = useState(false); // default false until API response
  const [UITeamHostExpiryWindow, setUITeamHostExpiryWindow] = useState<
    number | string
  >("");
  const [updatingTeamSettings, setUpdatingTeamSettings] = useState(false);
  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );
  const [addingCustomWindow, setAddingCustomWindow] = useState(false);

  const { renderFlash } = useContext(NotificationContext);

  const { isRouteOk, teamIdForApi } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: false,
      observer: false,
      observer_plus: false,
    },
  });

  const {
    data: appConfig,
    isLoading: isLoadingAppConfig,
    error: errorLoadGlobalConfig,
  } = useQuery<IConfig, Error, IConfig>(
    ["globalConfig"],
    () => configAPI.loadAll(),
    { refetchOnWindowFocus: false }
  );
  const {
    host_expiry_settings: {
      host_expiry_enabled: globalHostExpiryEnabled,
      host_expiry_window: globalHostExpiryWindow,
    },
  } = appConfig ?? { host_expiry_settings: {} };

  const {
    isLoading: isLoadingTeamConfig,
    refetch: refetchTeamConfig,
    error: errorLoadTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["teamConfig", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      enabled: isRouteOk && !!teamIdForApi,
      select: (data) => data.team,
      onSuccess: (teamConfig) => {
        // default this setting to current team setting
        // can be updated by user actions
        setUITeamHostExpiryEnabled(
          teamConfig?.host_expiry_settings?.host_expiry_enabled ?? false
        );
        setUITeamHostExpiryWindow(
          teamConfig.host_expiry_settings?.host_expiry_window ?? ""
        );
      },
      refetchOnWindowFocus: false,
    }
  );

  const validate = useCallback(() => {
    const errors: Record<string, string> = {};
    if (
      (!globalHostExpiryEnabled &&
        UITeamHostExpiryEnabled &&
        !UITeamHostExpiryWindow) ||
      Number(UITeamHostExpiryWindow) < 0
    ) {
      errors.host_expiry_window = HOST_EXPIRY_ERROR_TEXT;
    }

    setFormErrors(errors);
  }, [
    UITeamHostExpiryEnabled,
    UITeamHostExpiryWindow,
    globalHostExpiryEnabled,
  ]);

  useEffect(() => {
    validate();
  }, [UITeamHostExpiryEnabled, UITeamHostExpiryWindow, validate]);
  const updateTeamHostExpiry = useCallback(
    (evt: React.MouseEvent<HTMLFormElement>) => {
      evt.preventDefault();

      setUpdatingTeamSettings(true);
      teamsAPI
        .update(
          {
            host_expiry_settings: {
              host_expiry_enabled: UITeamHostExpiryEnabled,
              host_expiry_window: Number(UITeamHostExpiryWindow),
            },
          },
          teamIdForApi
        )
        .then(() => {
          renderFlash("success", "Successfully updated settings.");
          refetchTeamConfig();
        })
        .catch((errorResponse: { data: IApiError }) => {
          renderFlash(
            "error",
            `Could not update team settings. ${errorResponse.data.errors[0].reason}`
          );
        })
        .finally(() => {
          setUpdatingTeamSettings(false);
        });
    },
    [
      UITeamHostExpiryEnabled,
      UITeamHostExpiryWindow,
      refetchTeamConfig,
      renderFlash,
      teamIdForApi,
    ]
  );

  const renderForm = () => {
    if (errorLoadGlobalConfig || errorLoadTeamConfig) {
      return <DataError />;
    }
    if (isLoadingTeamConfig || isLoadingAppConfig) {
      return <Spinner />;
    }
    return (
      <form onSubmit={updateTeamHostExpiry}>
        {globalHostExpiryEnabled !== undefined &&
          globalHostExpiryWindow !== undefined && (
            <TeamHostExpiryToggle
              globalHostExpiryEnabled={globalHostExpiryEnabled}
              globalHostExpiryWindow={globalHostExpiryWindow}
              teamExpiryEnabled={UITeamHostExpiryEnabled}
              setTeamExpiryEnabled={setUITeamHostExpiryEnabled}
              addingCustomWindow={addingCustomWindow}
              setAddingCustomWindow={setAddingCustomWindow}
            />
          )}
        {(UITeamHostExpiryEnabled || addingCustomWindow) && (
          <InputField
            label="Host expiry window"
            type="number"
            onChange={setUITeamHostExpiryWindow}
            name="host-expiry-window"
            value={UITeamHostExpiryWindow}
            error={formErrors.host_expiry_window}
          />
        )}
        <Button
          type="submit"
          variant="brand"
          className="button-wrap"
          isLoading={updatingTeamSettings}
          disabled={Object.keys(formErrors).length > 0}
        >
          Save
        </Button>
      </form>
    );
  };

  return (
    <section className={`${baseClass}`}>
      <div className="section-header">Host expiry settings</div>
      {renderForm()}
    </section>
  );
};
export default TeamSettings;
