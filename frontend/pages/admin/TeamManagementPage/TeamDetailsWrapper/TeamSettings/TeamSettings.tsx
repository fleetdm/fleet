import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import useTeamIdParam from "hooks/useTeamIdParam";
import { ITeamConfig } from "interfaces/team";
import { ITeamSubnavProps } from "interfaces/team_subnav";
import React, { useCallback, useContext, useState } from "react";
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

const TeamSettings = ({ location, router }: ITeamSubnavProps) => {
  // encompasses both when global setting is not enabled and
  // when it is and the user has opted to override it with a local setting
  const [UITeamHostExpiryEnabled, setUITeamHostExpiryEnabled] = useState(false); // default false until API response
  const [UITeamHostExpiryWindow, setUITeamHostExpiryWindow] = useState<
    number | null
  >(null);
  const [updatingTeamSettings, setUpdatingTeamSettings] = useState(false);
  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );

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
    isLoading: isLoadingTeamData,
    refetch: refetchTeamConfig,
    error: errorLoadTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team_details", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      enabled: isRouteOk && !!teamIdForApi,
      select: (data) => data.team,
      onSuccess: (teamData) => {
        // default this setting to current team setting
        // can be updated by user actions
        setUITeamHostExpiryEnabled(
          teamData?.host_expiry_settings?.host_expiry_enabled ?? false
        );
      },
      refetchOnWindowFocus: false,
    }
  );

  const onExpiryWindowChange = (value: number) => {
    setUITeamHostExpiryWindow(value);
    // TODO - validate either  here or as effect
    // validate(value, expiryWindowErrorCondition, expiryWindowErrorMessage);
  };

  const updateTeamHostExpiry = useCallback(
    (evt: React.MouseEvent<HTMLFormElement>) => {
      evt.preventDefault();
      // TODO validate, here or as effect

      setUpdatingTeamSettings(true);
      teamsAPI
        .update(
          {
            host_expiry_settings: {
              host_expiry_enabled: UITeamHostExpiryEnabled,
              host_expiry_window: UITeamHostExpiryWindow ?? 0,
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
    ]
  );

  const renderForm = () => {
    if (errorLoadGlobalConfig || errorLoadTeamConfig) {
      return <DataError />;
    }
    if (isLoadingTeamData || isLoadingAppConfig) {
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
            />
          )}
        {UITeamHostExpiryEnabled && (
          <InputField
            label="Host expiry window"
            onChange={onExpiryWindowChange}
            name="host-expiry-window"
            value={UITeamHostExpiryWindow}
          />
        )}
        <Button
          type="submit"
          variant="brand"
          className="button-wrap"
          isLoading={updatingTeamSettings}
        >
          Save
        </Button>
      </form>
    );
  };

  return (
    <section className={`${baseClass}`}>
      <div className="section-header">Settings</div>
      {renderForm()}
    </section>
  );
};
export default TeamSettings;
